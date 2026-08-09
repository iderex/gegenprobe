package version

import "testing"

// The offending test. It imports testing and nothing else, so every rule the
// leg named `gate tier capabilities` reads source for is satisfied, and it
// still opens a socket the moment it runs.
//
// Where the network is reachable it passes and says nothing. Where it is not,
// it fails, and the only thing separating those two runs is the place the tier
// ran in.
func TestSomethingThatQuietlyNeedsTheNetwork(t *testing.T) {
	if err := Reach(); err != nil {
		t.Fatalf("this test needs the network and did not get it: %v", err)
	}
}
