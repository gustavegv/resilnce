package training

import (
	"context"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func lockSession(ctx context.Context, tx pgx.Tx, owner, id, expected string) (string, error) {
	if !validID(id) {
		return "", invalid("sessionId", "Use a UUID session ID")
	}
	n, err := strconv.ParseInt(expected, 10, 64)
	if err != nil || n < 1 || strconv.FormatInt(n, 10) != expected {
		return "", invalid("expectedRevision", "Use the reviewed positive revision string")
	}
	var revision string
	if err := tx.QueryRow(ctx, `SELECT revision::text FROM public.training_session WHERE owner_mail=$1 AND id=$2 FOR UPDATE`, owner, id).Scan(&revision); err != nil {
		return "", err
	}
	if revision != expected {
		return "", conflict("stale-revision", "The session changed. Review the current plan before saving")
	}
	return revision, nil
}

func (s *Service) UpdateSession(ctx context.Context, owner, id string, c UpdateSession) (out SessionResult, err error) {
	defer func() { err = storageError(err) }()
	command, err := s.beginCommand(ctx, owner, c.OperationID, "update-session", id, c)
	if err != nil {
		return out, err
	}
	defer command.rollback(ctx)
	tx := command.tx
	if command.replay {
		out.Session, err = readSession(ctx, tx, owner, id)
		if err != nil {
			return out, err
		}
		err = command.commit(ctx, "", "")
		out.Operation = command.operation
		return out, err
	}
	if err = validateSession(CreateSession{OperationID: c.OperationID, Name: c.Name, Items: c.Items}, true); err != nil {
		return out, err
	}
	if _, err = lockSession(ctx, tx, owner, id, c.ExpectedRevision); err != nil {
		return out, err
	}
	current, err := readSession(ctx, tx, owner, id)
	if err != nil {
		return out, err
	}
	retained := map[string]Placement{}
	for _, p := range current.Placements {
		retained[p.ID] = p
	}
	additions := []SessionItem{}
	seenPlacements := map[string]bool{}
	for _, item := range c.Items {
		if item.Kind != "retainPlacement" {
			additions = append(additions, item)
			continue
		}
		if _, ok := retained[item.PlacementID]; !ok {
			return out, &Error{404, "not-found", "Placement is unavailable in this session", nil}
		}
		if seenPlacements[item.PlacementID] {
			return out, invalid("items", "Do not repeat placements")
		}
		seenPlacements[item.PlacementID] = true
	}
	if err = lockSessionItems(ctx, tx, owner, additions); err != nil {
		return out, err
	}
	// The schema explicitly permits transactional position swaps. Defer only
	// this constraint and assign final positions without a temporary offset.
	if _, err = tx.Exec(ctx, `SET CONSTRAINTS public.session_variant_session_id_position_key DEFERRED`); err != nil {
		return out, err
	}
	for _, p := range current.Placements {
		if !seenPlacements[p.ID] {
			if _, err = tx.Exec(ctx, `DELETE FROM public.session_variant WHERE owner_mail=$1 AND id=$2`, owner, p.ID); err != nil {
				return out, err
			}
		}
	}
	seenVariants := map[string]bool{}
	changedExercises := map[string]bool{}
	for i, item := range c.Items {
		placementID, v := item.PlacementID, item.VariantID
		if item.Kind == "retainPlacement" {
			v = retained[placementID].VariantID
		} else if item.Kind != "reuseVariant" {
			e := item.ExerciseID
			if item.Kind == "createExercise" {
				e = newID()
				codes, _ := normalizeCodes(item.MuscleGroupCodes)
				if _, err = tx.Exec(ctx, `INSERT INTO public.exercise(id,owner_mail,name,muscle_group_codes) VALUES($1,$2,$3,$4)`, e, owner, strings.TrimSpace(item.Name), codes); err != nil {
					return out, err
				}
			} else {
				changedExercises[e] = true
			}
			name := strings.TrimSpace(item.Name)
			if item.Kind == "createExercise" {
				name = "Standard"
			}
			v, err = createVariant(ctx, tx, owner, e, name, item.Configuration)
			if err != nil {
				return out, err
			}
		}
		if seenVariants[v] {
			return out, invalid("items", "A variant can appear only once in a session")
		}
		seenVariants[v] = true
		if item.Kind == "retainPlacement" {
			_, err = tx.Exec(ctx, `UPDATE public.session_variant SET position=$3 WHERE owner_mail=$1 AND id=$2`, owner, placementID, i+1)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO public.session_variant(id,owner_mail,session_id,variant_id,position) VALUES($1,$2,$3,$4,$5)`, newID(), owner, id, v, i+1)
		}
		if err != nil {
			return out, err
		}
	}
	for e := range changedExercises {
		if _, err = tx.Exec(ctx, `UPDATE public.exercise SET revision=revision+1,updated_at=now() WHERE owner_mail=$1 AND id=$2`, owner, e); err != nil {
			return out, err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE public.training_session SET name=$3,revision=revision+1,updated_at=now() WHERE owner_mail=$1 AND id=$2`, owner, id, strings.TrimSpace(c.Name)); err != nil {
		return out, err
	}
	out.Session, err = readSession(ctx, tx, owner, id)
	if err != nil {
		return out, err
	}
	err = command.commit(ctx, id, out.Session.Revision)
	out.Operation = command.operation
	return out, err
}

func (s *Service) DeleteSession(ctx context.Context, owner, id string, c DeleteSession) (out DeleteSessionResult, err error) {
	defer func() { err = storageError(err) }()
	command, err := s.beginCommand(ctx, owner, c.OperationID, "delete-session", id, c)
	if err != nil {
		return out, err
	}
	defer command.rollback(ctx)
	if !command.replay {
		revision, err := lockSession(ctx, command.tx, owner, id, c.ExpectedRevision)
		if err != nil {
			return out, err
		}
		var active bool
		if err = command.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM public.workout WHERE owner_mail=$1 AND source_session_id=$2 AND status='active')`, owner, id).Scan(&active); err != nil {
			return out, err
		}
		if active {
			return out, conflict("active-source-session", "Finish or quit the active workout before deleting its source session")
		}
		if _, err = command.tx.Exec(ctx, `DELETE FROM public.training_session WHERE owner_mail=$1 AND id=$2`, owner, id); err != nil {
			return out, err
		}
		if err = command.commit(ctx, id, revision); err != nil {
			return out, err
		}
	} else if err = command.commit(ctx, "", ""); err != nil {
		return out, err
	}
	return DeleteSessionResult{Operation: command.operation, SessionID: id, Deleted: true}, nil
}
