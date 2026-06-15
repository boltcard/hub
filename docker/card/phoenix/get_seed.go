package phoenix

import (
	"fmt"
	"os"
	"strings"
)

// seedFilePath is the location of the phoenixd seed file inside the card
// container. The phoenix data volume is mounted read-only at /root/.phoenix
// (see docker-compose.yml), and phoenixd writes the wallet's BIP39 mnemonic
// here as a single line of space-separated words in plaintext. It is a var
// (not a const) so tests can point it at a temporary file.
var seedFilePath = "/root/.phoenix/seed.dat"

// GetSeedWords reads the phoenixd wallet recovery phrase from seed.dat and
// returns it as a slice of individual words. The file is read fresh on each
// call (never cached) so the secret does not linger in process memory.
func GetSeedWords() ([]string, error) {
	data, err := os.ReadFile(seedFilePath)
	if err != nil {
		return nil, fmt.Errorf("read phoenix seed file: %w", err)
	}

	words := strings.Fields(string(data))
	if len(words) == 0 {
		return nil, fmt.Errorf("phoenix seed file is empty")
	}

	return words, nil
}
