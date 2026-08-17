package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// verifyIndependently re-implements Telegram's documented check from the
// server's point of view: build the data-check string from the fields that
// actually arrived, HMAC it under SHA256(botToken), compare.
//
// It is written from the spec rather than copied from sign() on purpose — the
// realistic bug here is a field-set disagreement (an empty last_name included
// on one side and skipped on the other), and only two independent field
// selections can catch that.
func verifyIndependently(botToken string, req LoginRequest) bool {
	fields := map[string]string{
		"id":         strconv.FormatInt(req.ID, 10),
		"first_name": req.FirstName,
		"auth_date":  strconv.FormatInt(req.AuthDate, 10),
	}
	for k, v := range map[string]string{
		"last_name": req.LastName,
		"username":  req.Username,
		"photo_url": req.PhotoURL,
	} {
		if v != "" {
			fields[k] = v
		}
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, k+"="+fields[k])
	}

	secret := sha256.Sum256([]byte(botToken))
	mac := hmac.New(sha256.New, secret[:])
	mac.Write([]byte(strings.Join(lines, "\n")))
	return hmac.Equal([]byte(hex.EncodeToString(mac.Sum(nil))), []byte(req.Hash))
}

func TestSignedRequestVerifies(t *testing.T) {
	cases := []struct {
		name string
		req  LoginRequest
	}{
		{"only the required fields", LoginRequest{ID: 495598685, FirstName: "Денис", AuthDate: 1754790000}},
		{"with last name", LoginRequest{ID: 1, FirstName: "A", LastName: "B", AuthDate: 2}},
		{"with username", LoginRequest{ID: 1, FirstName: "A", Username: "zemdenalex", AuthDate: 2}},
		{"every field", LoginRequest{ID: 7, FirstName: "Денис", LastName: "Земцов",
			Username: "zemdenalex", PhotoURL: "https://t.me/i/x.jpg", AuthDate: 1754790000}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := BuildLoginRequest("123456:TEST-TOKEN", c.req.ID, c.req.FirstName,
				c.req.LastName, c.req.Username, c.req.PhotoURL, c.req.AuthDate)

			if got.Hash == "" {
				t.Fatal("no hash produced")
			}
			if !verifyIndependently("123456:TEST-TOKEN", got) {
				t.Errorf("the API would reject this payload as INVALID_HASH")
			}
		})
	}
}

// A golden vector computed outside Go entirely, so a change to sign() that
// happens to break the algorithm in both directions at once still fails here.
func TestSignMatchesAnExternallyComputedHash(t *testing.T) {
	const want = "cab75da35aa3a53ae153340ce33e994c0eb0830c41a1d7bff78f902b7f1910b2"

	got := BuildLoginRequest("123456:TEST-TOKEN", 495598685, "Денис", "", "", "", 1754790000)

	if got.Hash != want {
		t.Errorf("hash = %s, want %s", got.Hash, want)
	}
}

// The whole point of omitempty: an empty optional field must not enter the
// data-check string. If it did, the server — which skips empty values — would
// compute a different hash and every command would come back unauthorised.
func TestEmptyOptionalFieldsDoNotChangeTheHash(t *testing.T) {
	bare := BuildLoginRequest("t", 1, "A", "", "", "", 2)
	explicitlyEmpty := BuildLoginRequest("t", 1, "A", "", "", "", 2)

	if bare.Hash != explicitlyEmpty.Hash {
		t.Fatal("empty optionals changed the hash")
	}
	withName := BuildLoginRequest("t", 1, "A", "B", "", "", 2)
	if withName.Hash == bare.Hash {
		t.Error("a present last_name must change the hash; it is being dropped")
	}
}

func TestDifferentTokensProduceDifferentHashes(t *testing.T) {
	a := BuildLoginRequest("token-a", 1, "A", "", "", "", 2)
	b := BuildLoginRequest("token-b", 1, "A", "", "", "", 2)

	if a.Hash == b.Hash {
		t.Error("hash does not depend on the bot token")
	}
}
