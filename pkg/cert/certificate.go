package cert

type Certificate struct {
	Names []string
	Cert  []byte
	Key   []byte
}
