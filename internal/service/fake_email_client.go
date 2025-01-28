package service

import (
	"crypto/md5"
	"fmt"
	"math/big"
)

// FakeEmailClient is a simple fake implementation of RawEmailClient
type FakeEmailClient struct{}

func (f *FakeEmailClient) SendRawEmail(input *RawEmailInput) (*RawEmailOutput, error) {
	// Hash the raw input data using MD5
	hash := md5.New()
	hash.Write(input.Data)
	hashBytes := hash.Sum(nil)

	// Convert the first byte to an integer (to analyze its first character)
	hashInt := big.NewInt(0)
	hashInt.SetBytes(hashBytes)

	// Convert the integer to a string to check the first character
	hashString := fmt.Sprintf("%x", hashInt)

	// Check the first character of the hash to simulate success or failure
	firstChar := hashString[0]

	// If the first character is a letter (a-z) or an even number (0, 2, 4, 6, 8), simulate failure
	if (firstChar >= 'a' && firstChar <= 'z') || (firstChar >= '0' && firstChar <= '9' && firstChar%2 == 0) {
		return nil, fmt.Errorf("operation error FAKE: SendRawEmail, simulated failure")
	}

	// Simulate a success with a fake message ID if the first character is an odd number
	return &RawEmailOutput{
		MessageID: "fake-message-id", // Fake message ID for success
	}, nil
}
