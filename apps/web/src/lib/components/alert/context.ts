import { getContext, setContext } from 'svelte';

export type AlertDialogContext = {
  open: boolean;
  setOpen: (value: boolean) => void;
  close: () => void;
};

const ALERT_DIALOG_CONTEXT = Symbol('ALERT_DIALOG_CONTEXT');

export function setAlertDialogContext(value: AlertDialogContext): AlertDialogContext {
  setContext(ALERT_DIALOG_CONTEXT, value);
  return value;
}

export function getAlertDialogContext(): AlertDialogContext {
  return getContext<AlertDialogContext>(ALERT_DIALOG_CONTEXT);
}
