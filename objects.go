package main

import (
	"os"
	"time"
)

//PORT port to be used
const PORT = "8080"
const hashCost = 8
const (
	host     = "localhost"
	port     = 5432
	user     = "postgres"
	password = "postgres"
	dbname   = "postgres"
)

// Nome do arquivo de log
var logFile, _ = os.OpenFile("log_"+time.Now().Format("01-02-2006")+".log", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0660)

// PF struct que guarda a pessoa fisica
type PF struct {
	Nome         string `json:"nm_pessoa_fisica"`
	CPF          string `json:"CPF"`
	DtNascimento string `json:"dt_nascimento"`
}

// FuncPublico guarda o funcionario antes de importar para o banco
type FuncPublico struct {
	Name     string
	Position string
	Place    string
	Salary   float64
}

// User guarda o usuário antes de importar para o banco
type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Position string `json:"position"`
}

// Credentials estrutura para fazer login
type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}

// Users estrutura que guarda os dados da tabela users
type Users struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Clientes estrutura que guarda os dados da tabela clients
type Clientes struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Salary   float64 `json:"salary"`
	Position string  `json:"position"`
	Place    string  `json:"place"`
	IsClient string  `json:"isclient"`
}