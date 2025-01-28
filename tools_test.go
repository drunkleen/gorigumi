package toolkit

import "testing"

func TestTools_GenerateRandomString(t *testing.T) {
	var testTools Tools
	randString := testTools.GenerateRandomString(10)
	if len(randString) != 10 {
		t.Error("Expected string of length 10, but got", len(randString))
	}
	randString1, randString2 := testTools.GenerateRandomString(10), testTools.GenerateRandomString(10)
	if randString1 == randString2 {
		t.Error("Expected random strings to be different")
	}

}
