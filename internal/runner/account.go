package runner

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// Account is the 12-digit fake AWS account an exercise runs in. Floci uses a
// 12-digit access key ID as the account ID and keeps every account's resources
// apart, so exercises never see each other's. It depends only on the exercise
// name and its reset counter: the same exercise always gets the same account
// until it is reset, and a reset gets a fresh, empty one.
func Account(exercise string, resets int) string {
	sum := sha256.Sum256(fmt.Appendf(nil, "tflings/%s/%d", exercise, resets))
	n := binary.BigEndian.Uint64(sum[:8])
	// 100000000000..999999999999: never starts with 0, so it cannot be the
	// default account 000000000000 or one of the spikes' small numbers.
	return fmt.Sprintf("%012d", 100_000_000_000+n%900_000_000_000)
}
