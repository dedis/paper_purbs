package purbs

import (
	"crypto/cipher"
	"github.com/dedis/kyber"
	"github.com/dedis/kyber/util/key"
)

// Suite defines the required functionalities for each suite from kyber
type Suite interface {
	kyber.Encoding
	kyber.Group
	kyber.HashFactory
	kyber.XOFFactory
	kyber.Random
}

// A map of suite->info, info being the length of a marshalled public key, and the Allowed Positions in the purb header
type SuiteInfoMap map[string]*SuiteInfo

// SuiteInfo holds possible positions whose cornerstones might take in a header
// and a PayloadKey length for this suite
type SuiteInfo struct {
	AllowedPositions  []int // alternative PayloadKey/point position in purb header
	CornerstoneLength int   // length of each PayloadKey/point in bytes
}

// Structure to define the whole PURB
type Purb struct {
	Nonce []byte // Nonce used in both AEAD of entrypoints and payload. The same for different entrypoints
	// as the keys are different. It is stored in the very beginning of the purb
	Header     *Header
	Payload    []byte // Payload contains already padded plaintext
	PayloadKey []byte // Payload PayloadKey

	isVerbose bool // If true, the various operations on the data structure will print what is happening

	recipients      []Recipient                // tuple with (Suite, PublicKey, PrivateKey)
	infoMap         SuiteInfoMap               // public suite information (Allowed Positions, etc)
	symmKeyWrapType SYMMETRIC_KEY_WRAPPER_TYPE // type of encryption for the Entrypoints (symmetric or AEAD)
	stream          cipher.Stream              // Used to get randomness

	originalData []byte // Kept to compare between "Payload" and this
}

// Structure defining the actual header of a purb
type Header struct {
	EntryPoints      map[string][]*EntryPoint // map of suiteName -> []entrypoints
	Cornerstones     map[string]*Cornerstone  // Holds sender's ephemeral private/public keys for each suite in the header
	Layout           SkipLayout               // An array of byte slices where each of the bytes slice represents a hash table entry
	Length           int                      //
	EntryPointLength int                      // Length of each encrypted entry point
}

// Ephemeral Diffie-Hellman keys for all PayloadKey-holders using this suite.
// Should have a uniform representation, e.g., an Elligator point.
type Cornerstone struct {
	SuiteName string
	KeyPair   *key.Pair
	Offset    int    // Starting byte position in the header
	Bytes     []byte // singleton. Since calling marshalling the KeyPair is non-deterministic, at least we do it only once so prints are consistents
}

//EntryPoint holds the info required to create an entrypoint for each recipient.
type EntryPoint struct {
	Recipient    Recipient // Recipient whom this entrypoint is for
	SharedSecret []byte    // Ephemeral secret derived from negotiated DH secret
	Offset       int       // Starting byte position in the header
}

// Recipient holds information needed to be able to encrypt anything for it
// PrivateKey is nil for encoder
type Recipient struct {
	SuiteName string
	Suite
	PublicKey  kyber.Point
	PrivateKey kyber.Scalar
}
