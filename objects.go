package main

// PF struct que guarda a pessoa fisica
type PF struct {
	Nome         string `json:"nm_pessoa_fisica"`
	CPF          string `json:"CPF"`
	DtNascimento string `json:"dt_nascimento"`
}
