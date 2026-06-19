package app

import (
	"fmt"
	"strings"
)

var spaceAdjectives = []string{
	"amber", "ancient", "astral", "binary", "bright", "calm", "celestial", "cosmic",
	"crimson", "distant", "eclipse", "frozen", "galactic", "glowing", "golden", "hidden",
	"ion", "lunar", "magnetic", "midnight", "nebula", "nova", "orbital", "pale",
	"polar", "quantum", "radiant", "rogue", "silent", "solar", "stellar", "tidal",
	"umbra", "velvet", "violet", "void", "wandering", "zenith",
}

var spaceNouns = []string{
	"asteroid", "aurora", "beacon", "comet", "crater", "dawn", "dust", "equinox",
	"flare", "galaxy", "halo", "horizon", "meteor", "moon", "nebula", "nova",
	"orbit", "parallax", "planet", "pulsar", "quasar", "ray", "rocket", "satellite",
	"signal", "solstice", "star", "station", "sun", "tail", "telescope", "transit",
	"umbra", "vector", "voyager", "zenith",
}

func generateSlug() (string, error) {
	adj, err := randomChoice(spaceAdjectives)
	if err != nil {
		return "", err
	}
	noun, err := randomChoice(spaceNouns)
	if err != nil {
		return "", err
	}
	number, err := randomBase62Number(2)
	if err != nil {
		return "", err
	}
	return strings.ToLower(fmt.Sprintf("%s-%s-%s", adj, noun, number)), nil
}

func randomChoice(values []string) (string, error) {
	token, err := randomBase62(6)
	if err != nil {
		return "", err
	}
	var n int
	for i := range token {
		n += int(token[i])
	}
	return values[n%len(values)], nil
}

func randomBase62Number(length int) (string, error) {
	return randomBase62(length)
}
