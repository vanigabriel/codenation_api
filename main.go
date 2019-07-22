package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	//_ "./docs/docs.go" // For gin-swagger

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
)

func setupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	// Loga para arquivo
	gin.DefaultWriter = io.MultiWriter(logFile, os.Stdout)

	// Requisição do login inicial
	r.POST("/login", Login)

	// Para acessar esse grupo, precisa enviar na requisição o user: admin e pass: admin, modelo basic auth
	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		"admin": "admin",
	}))

	// Rotas dos usuários
	authorized.GET("users", getUsers)
	authorized.POST("users", registerUser)
	authorized.PUT("users", updateUser)
	authorized.DELETE("users/:id", deleteUser)

	authorized.POST("clients", uploadCliente)
	authorized.GET("clients", getClientes) // Get Clientes

	// Get notificações

	// Funcionarios publicos dos ultimos meses
	// Verificar Docker
	// DAshboard : Qtd total de +20mil, qtd q eu detenho, valor, dos ultimos meses, o mairo valor de salário, menor valor de salário, média, qtd de pessoa por orgão

	// Rota para criar administrador
	authorized.POST("admin", registerAdministrator)

	return r
}

func main() {

	handleError(godotenv.Load()) // Load env variables

	// Configura log para arquivo
	wrt := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(wrt)
	defer logFile.Close()

	// A cada 24hrs ele vai disparar a função que baixa e importa o CSV dos funcionários publicos de SP
	ticker := time.NewTicker(24 * time.Hour)
	go func() {
		// Realiza uma execução antes de começar o contador
		baixarCSV()
		for range ticker.C {
			baixarCSV()
		}
	}()

	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")

}

// Registra usuário no banco (que irá receber o alerta)
func registerUser(c *gin.Context) {
	usrs := &Users{}
	// Junta JSON com a struct
	c.BindJSON(&usrs)

	// Valida e-mail
	if usrs.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados obrigatórios não recebidos"})
	}

	// Abre conexão com o banco
	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer db.Close()

	// Valida se já existe usuário com esse email
	if rowExists("SELECT id FROM users WHERE email=$1", db, usrs.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email já cadastrado"})
	}

	// Insere
	_, err = db.Exec("INSERT INTO users (name, email, created_on) VALUES ($1, $2, now())", usrs.Name, usrs.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	// Retorna OK
	c.JSON(http.StatusOK, gin.H{"message": "Usuário inserido"})
}

// Login recebe um JSON com o usuario e senha do banco e valida os mesmos
func Login(c *gin.Context) {
	creds := &Credentials{}
	c.BindJSON(&creds)

	// Valida se a msg está correta
	if creds.Username == "" || creds.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados obrigatórios não recebidos"})
	}

	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer db.Close()

	// Recupera senha
	row := db.QueryRow("SELECT password FROM administrators WHERE username=$1", creds.Username)

	storedCreds := &Credentials{}
	err = row.Scan(&storedCreds.Password) // guardando a passw para comparar
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "usuário não encontrado."})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	// Criptografa a senha informada e compara com a do banco
	err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password))

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Senha incorreta"})
		return
	}

	//se passou, passamos o usuario e senha
	c.JSON(http.StatusOK, gin.H{"user": "admin", "pass": "admin"})
}

// Registra administrador
func registerAdministrator(c *gin.Context) {
	creds := &Credentials{}
	c.BindJSON(&creds)

	hashpwd, err := bcrypt.GenerateFromPassword([]byte(creds.Password), hashCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Problemas para criptografar a senha."})
	}

	db, err := initDB()
	_, err = db.Exec("INSERT INTO administrators (username, password, created_on) VALUES ($1, $2, now())", creds.Username, string(hashpwd))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
}

// Recupera usuários que receberão os e-mails
func getUsers(c *gin.Context) {
	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name, email FROM users ORDER BY name")
	defer rows.Close()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	usrs := []User{}

	for rows.Next() {
		usr := new(User)
		rows.Scan(&usr.ID, &usr.Name, &usr.Email)
		usrs = append(usrs, *usr)
	}
	c.JSON(http.StatusOK, usrs)
}

// Recupera os clientes
func getClientes(c *gin.Context) {
	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer db.Close()

	rows, err := db.Query("SELECT id, name, salary, position, place, case when is_special = True then 'yes' else 'no' end as is_special FROM clients ORDER BY name")
	defer rows.Close()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	clients := []Clientes{}

	for rows.Next() {
		client := new(Clientes)
		rows.Scan(&client.ID, &client.Name, &client.Salary, &client.Position, &client.Place, &client.IsClient)
		clients = append(clients, *client)
	}
	c.JSON(http.StatusOK, clients)
}

// Faz upload dos clientes (arquivo csv)
func uploadCliente(c *gin.Context) {
	// Capturando arquivo com o ID file
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	// Salvando arquivo localmente
	err = c.SaveUploadedFile(file, "uploadClientes.csv")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	// Abrindo arquivo salvo
	f, err := os.Open("uploadClientes.csv")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer f.Close()

	//Abrindo conexão com o banco de dados
	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer db.Close()

	// Abre transação
	trc, _ := db.Begin()

	// Inicia leitura do CSV
	r := csv.NewReader(bufio.NewReader(f))

	//Iterando pelo arquivo
	for {
		record, err := r.Read()

		// Se fim de arquivo
		if err == io.EOF {
			break
		}

		// Se já existe aquele cliente, não faz nada
		if rowExists("SELECT id FROM clients WHERE name=$1", db, record[0]) {
			continue
		}

		// Insere cliente
		_, err = trc.Exec("INSERT INTO clients (name, created_on) VALUES ($1, now())", record[0])
		if err != nil {
			trc.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		}
	}

	// Commit transação
	err = trc.Commit()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Clientes inseridos."})
}

func updateUser(c *gin.Context) {
	id := c.Param("id")
	var usrs User
	err := c.BindJSON(&usrs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados inconsistentes"})
		return
	}

	//Abrindo conexão com o banco de dados
	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer db.Close()

	res, err := db.Exec("UPDATE users SET name=$2, email=$3,position=$4 WHERE id=$1", id, usrs.Name, usrs.Email, usrs.Position)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}

	rows, err := res.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Nenhum usuário alterado."})
	}

	c.JSON(200, gin.H{"message": "Usuário alterado com sucesso"})

}

func deleteUser(c *gin.Context) {
	id := c.Param("id")

	//Abrindo conexão com o banco de dados
	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	defer db.Close()

	res, err := db.Exec("DELETE FROM users WHERE id=$1", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Não foi possível conectar ao BD"})
	}

	rows, err := res.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
	}
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Nenhum usuário removido"})
	}

	c.JSON(200, gin.H{"message": "Usuário removido com sucesso"})

}
