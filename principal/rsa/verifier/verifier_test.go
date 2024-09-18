package verifier

import "testing"

func TestParse(t *testing.T) {
	str := "did:key:z4MXj1wBzi9jUstyNgxg2TNN9cNWH8BzcMa5iZ9DAUiLutvQPgBu3zE385tUsbd4oVfHwFb2afSmHpKG4x8JVzESNPSCri4fgztu9FdV3FArz2gByZ9E6zKk3snQKuRjfMJTf29b4BLwGu9j7BtJnhR7bWDWvNqo2YSAwEP8UXyV1W7Meiu96v4esmv2sBLug4vkMFDKXx8bdYZNJYGQQHYrqGXRStZZYGK9xiddMutKeopr1q9UKrczbFhWbdsHW587y4p4uVfwj8evGak6Gx7ADHyQPJc5jWmmUXTzZHJwTqEXDekFkQwkfR9ycxWKnSmPcN9mnimKmuD4LMMzZbodM8Ukgo7XGW8HbiUf3utjt6carBD4c"
	v, err := Parse(str)
	if err != nil {
		t.Fatalf("parsing DID: %s", err)
	}
	if v.DID().String() != str {
		t.Fatalf("expected %s to equal %s", v.DID().String(), str)
	}
}