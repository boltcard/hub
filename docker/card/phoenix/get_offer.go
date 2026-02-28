package phoenix

func GetOffer() (string, error) {
	body, err := doGet("/getoffer", "GetOffer")
	if err != nil {
		return "", err
	}

	return string(body), nil
}
