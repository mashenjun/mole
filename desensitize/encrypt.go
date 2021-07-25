package desensitize

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
)

type AESEncrypt struct {
	key       []byte
	block     cipher.Block
	blockMode cipher.BlockMode
	cache     map[string]string
}

func NewAESEncrypt(key []byte) (*AESEncrypt, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockMode := cipher.NewCBCEncrypter(block, key[:block.BlockSize()])
	return &AESEncrypt{block: block, blockMode: blockMode, key: key, cache: make(map[string]string)}, nil
}

func (e *AESEncrypt) Encrypt(plain string) string {
	if v, ok := e.cache[plain]; ok {
		return v
	}
	origin := padding([]byte(plain), e.block.BlockSize())
	out := make([]byte, len(origin))
	// rebuild the block mode every time encrypt plain text
	// blockMode := cipher.NewCBCEncrypter(e.block, e.key[:e.block.BlockSize()])
	e.blockMode.CryptBlocks(out, origin)
	cs := hex.EncodeToString(out[:len(plain)])[:len(plain)]
	if _, ok := e.cache[plain]; !ok {
		e.cache[plain] = cs
	}
	return cs
}

func padding(plain []byte, blockSize int) []byte {
	padding := blockSize - len(plain)%blockSize
	if padding == blockSize {
		return plain
	}
	padtext := bytes.Repeat([]byte{'='}, padding)
	return append(plain, padtext...)
}
