package desensitize

import "testing"

func TestAESEncrypt_Encrypt(t *testing.T) {
	enc, err := NewAESEncrypt([]byte("myverystrongpasswordo32bitlength"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(enc.Encrypt("HELLO"))
	t.Log(enc.Encrypt("WORLD"))
	t.Log(enc.Encrypt("foo"))
	t.Log(enc.Encrypt("bar"))
	t.Log(enc.Encrypt("foo"))
	t.Log(enc.Encrypt("bar"))
}
