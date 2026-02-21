package phoenix

func GetOffer() (offer string, err error) {
	body, err := doGet("/getoffer", "GetOffer")
	if err != nil {
		return "", err
	}

	return string(body), nil
}
