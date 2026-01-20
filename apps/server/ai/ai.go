package ai

func GetQuickCreationData(promptSelections []bool, userData string) (string, error) {
	callPrompt, err := getPrompt(promptSelections, userData)
	if err != nil {
		return "", err
	}

	jsonString, err := callChatAPI(callPrompt)
	if err != nil {
		return "", err
	}

	return jsonString, nil
}
