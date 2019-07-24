package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"io"
	"log"
	"net/http"
	"os"

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

	// Usuários
	authorized.GET("users", getUsers)            // Recupera usuários
	authorized.POST("users", registerUser)       // Registra usuário
	authorized.PUT("users/:id", updateUser)      // Atualiza usuário :id
	authorized.DELETE("users/:id", inactiveUser) // Inativa usuário :id

	// Clientes
	authorized.POST("clients", uploadCliente) // Carrega clientes do arquivo
	authorized.GET("clients", getClientes)    // Get Clientes

	authorized.POST("publicagents", updatePublicAgents) //Atualiza funcionários publicos
	authorized.POST("events", sentEmail)                // Cria evento e envia e-mail

	// Get notificações

	/*
		Esquema notificações:
			Importa csv -> valida cliente

			Ao fim da importação do csv, ele irá fazer um join com a tabela de clientes, vendo quem ainda não é cliente e ganha +20mil
			com o resultado, ele irá inserir na tabela de eventos os que não tiveram e-mail enviado a mais de 7 dias
	*/

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

	// Toda segunda às 07:30 ele vai disparar a função que baixa e importa o CSV dos funcionários publicos de SP
	//gocron.Every(1).Monday().At("07:30").Do(schedulerAgents)
	//<-gocron.Start()

	r := setupRouter()

	r.Run(":" + os.Getenv("port"))

}

// Registra usuário no banco (que irá receber o alerta)
func registerUser(c *gin.Context) {
	log.Println("Registrando usuário")
	usrs := &Users{}
	// Junta JSON com a struct
	c.BindJSON(&usrs)

	log.Println("Validando se o e-mail foi recebido")
	// Valida e-mail
	if usrs.Email == "" {
		log.Println("tag email não recebido no json")
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados obrigatórios não recebidos"})
		return
	}

	log.Println("Abrindo conexão com o banco")
	// Abre conexão com o banco
	db, err := initDB()
	if err != nil {
		log.Println("Erro ao iniciar o banco")
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer db.Close()

	log.Println("Verificando se existe outro usuário ativo com esse e-mail")
	// Valida se já existe usuário ativo com esse email
	if rowExists("SELECT id FROM users WHERE email=$1 and is_active IS DISTINCT FROM 'N'", db, usrs.Email) {
		log.Println("Email já cadastrado")
		c.JSON(http.StatusBadRequest, gin.H{"message": "Email já cadastrado"})
		return
	}

	log.Println("Inserindo usuário")
	// Insere
	_, err = db.Exec("INSERT INTO users (name, email,position, created_on) VALUES ($1, $2,$3, now())", usrs.Name, usrs.Email, usrs.Position)
	if err != nil {
		log.Println("Erro ao inserir usuário")
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	log.Println("Usuário inserido com sucesso")
	// Retorna OK
	c.JSON(http.StatusCreated, gin.H{"message": "Usuário inserido"})
}

// Login recebe um JSON com o usuario e senha do banco e valida os mesmos
func Login(c *gin.Context) {
	log.Println("Iniciando login")
	creds := &Credentials{}
	c.BindJSON(&creds)

	// Valida se a msg está correta
	if creds.Username == "" || creds.Password == "" {
		log.Println("Dados obrigatórios não recebidos")
		c.JSON(http.StatusBadRequest, gin.H{"message": "Dados obrigatórios não recebidos"})
		return
	}

	db, err := initDB()
	if err != nil {
		log.Println("Erro ao iniciar o banco")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer db.Close()

	log.Println("Consulta no banco de dados")
	// Recupera senha
	row := db.QueryRow("SELECT password FROM administrators WHERE username=$1", creds.Username)

	storedCreds := &Credentials{}
	err = row.Scan(&storedCreds.Password) // guardando a passw para comparar
	if err != nil {
		if err == sql.ErrNoRows {
			log.Println("Usuário não encontrado")
			c.JSON(http.StatusUnauthorized, gin.H{"message": "usuário não encontrado."})
			return
		}
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	log.Println("Comparando senha")
	// Criptografa a senha informada e compara com a do banco
	err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password))

	if err != nil {
		log.Println("Senha incorreta")
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Senha incorreta"})
		return
	}

	log.Println("Autenticado")
	//se passou, passamos o usuario e senha
	c.JSON(http.StatusOK, gin.H{"user": "admin", "pass": "admin"})
}

// Registra administrador
func registerAdministrator(c *gin.Context) {
	log.Println("Iniciando registerAdministrator")

	creds := &Credentials{}
	c.BindJSON(&creds)

	hashpwd, err := bcrypt.GenerateFromPassword([]byte(creds.Password), hashCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "Problemas para criptografar a senha."})
		return
	}

	db, err := initDB()
	_, err = db.Exec("INSERT INTO administrators (username, password, created_on) VALUES ($1, $2, now())", creds.Username, string(hashpwd))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "administrator created"})
}

// Recupera usuários que receberão os e-mails
func getUsers(c *gin.Context) {
	log.Println("Iniciando getUsers")

	db, err := initDB()
	if err != nil {
		log.Println("Erro ao iniciar o banco de dados")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer db.Close()

	log.Println("Consultando usuários")
	rows, err := db.Query("SELECT id, name, email FROM users where is_active IS DISTINCT FROM 'N' ORDER BY name")
	if err != nil {
		log.Println("Erro ao consultar")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer rows.Close()

	usrs := []User{}

	for rows.Next() {
		usr := new(User)
		rows.Scan(&usr.ID, &usr.Name, &usr.Email)
		usrs = append(usrs, *usr)
	}

	log.Println("Consulta finalizando, retornando")

	c.JSON(http.StatusOK, usrs)
}

// Recupera os clientes
func getClientes(c *gin.Context) {
	log.Println("Iniciando getClients")

	db, err := initDB()
	if err != nil {
		log.Println("Erro ao abrir conexão com o banco de dados")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer db.Close()

	log.Println("Consultando clientes")

	rows, err := db.Query("SELECT id, name, salary, position, place, case when is_special = True then 'yes' else 'no' end as is_special FROM clients ORDER BY name")
	if err != nil {
		log.Println("Erro ao consultar")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer rows.Close()

	clients := []Clientes{}

	for rows.Next() {
		client := new(Clientes)
		rows.Scan(&client.ID, &client.Name, &client.Salary, &client.Position, &client.Place, &client.IsClient)
		clients = append(clients, *client)
	}

	log.Println("Retornando clientes")

	c.JSON(http.StatusOK, clients)
}

// Faz upload dos clientes (arquivo csv)
func uploadCliente(c *gin.Context) {
	log.Println("Iniciando uploadCliente")

	log.Println("Capturando arquivo 'file'")
	// Capturando arquivo com o ID file
	file, err := c.FormFile("file")
	if err != nil {
		log.Println("Arquivo não localizado")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "file not informed"})
		return
	}

	log.Println("Salvando arquivo localmento uploadClients.csv")
	// Salvando arquivo localmente
	err = c.SaveUploadedFile(file, "uploadClientes.csv")
	if err != nil {
		log.Println("Erro ao salvar arquivo")
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	log.Println("Carregando arquivo salvo")
	// Abrindo arquivo salvo
	f, err := os.Open("uploadClientes.csv")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer f.Close()

	//Abrindo conexão com o banco de dados
	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer db.Close()

	// Abre transação
	trc, _ := db.Begin()

	log.Println("Iniciando leitura CSV")
	// Inicia leitura do CSV
	r := csv.NewReader(bufio.NewReader(f))

	log.Println("Inserindo clientes")
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

		sql := `INSERT INTO clients (name, position, place, salary, lote_id, is_special, created_on)
					select 
					$1,
					b.position,
					b.place,
					b.salary,
					b.id_lote,
					case when b.salary >= 20000  then true 
					else false
					end,
					now()
				from now()
				left join public_agent b on b.name = $2`
		// Insere cliente
		_, err = trc.Exec(sql, record[0], record[0])
		if err != nil {
			trc.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": err})
			return
		}
	}

	// Commit transação
	err = trc.Commit()

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	log.Println("Clientes inseridos")
	c.JSON(http.StatusCreated, gin.H{"message": "Clientes inseridos."})
}

func updateUser(c *gin.Context) {
	log.Println("Iniciando updateUsers")
	id := c.Param("id")
	log.Println("Carregando ID")
	if len(id) == 0 {
		log.Println("ID não localizado")
		c.JSON(http.StatusBadRequest, gin.H{"message": "ID não informado"})
		return
	}
	var usrs User
	err := c.BindJSON(&usrs)
	if err != nil {
		log.Println("Erro ao dar o bind no Json")
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"message": "Erro ao decodificar JSON, verifique documentação dos tipos."})
		return
	}

	//Abrindo conexão com o banco de dados
	db, err := initDB()
	if err != nil {
		log.Println("Erro ao iniciar o banco de dados")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer db.Close()

	log.Println("Realizando Update")
	res, err := db.Exec("UPDATE users SET name=$1, email=$2,position=$3 WHERE id=$4", usrs.Name, usrs.Email, usrs.Position, id)
	if err != nil {
		log.Println("Erro ao realizar o update")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	rows, err := res.RowsAffected()
	if err != nil {
		log.Println("Erro ao realizar update")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	if rows == 0 {
		log.Println("Nenhum usuário alterado com o ID informado")
		c.JSON(http.StatusOK, gin.H{"message": "Nenhum usuário alterado com o ID informado."})
		return
	}

	log.Println("Usuário alterado com sucesso")
	c.JSON(200, gin.H{"message": "Usuário alterado com sucesso"})

}

// Inativa usuário
func inactiveUser(c *gin.Context) {
	id := c.Param("id")

	//Abrindo conexão com o banco de dados
	db, err := initDB()
	if err != nil {
		log.Println("Erro ao iniciar o banco de dados")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	defer db.Close()

	res, err := db.Exec("update users set is_active = 'N' WHERE id=$1", id)
	if err != nil {
		log.Println("Erro ao realizar update")
		log.Println(err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}

	rows, err := res.RowsAffected()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Nenhum usuário inativado"})
		return
	}

	c.JSON(200, gin.H{"message": "Usuário inativado com sucesso"})

}

func updatePublicAgents(c *gin.Context) {
	log.Println("Iniciando rotina de atualização dos funcionários publicos")
	if lockAgents != 'S' {
		go func() {
			err := baixarCSV()
			handleError(err)

		}()

		c.JSON(http.StatusOK, gin.H{"message": "Rotina iniciada em segundo plano"})
		return
	}
	log.Println("Rotina já em execução, abortado")
	c.JSON(http.StatusBadRequest, gin.H{"message": "Rotina já está executando em segundo plano, favor aguarde"})

}

func sentEmail(c *gin.Context) {
	log.Println("Iniciando rotina para criação e envio de eventos")

	go func() {
		err := createEvents("full")
		handleError(err)
		log.Println("Rotina finalizada")
	}()

	c.JSON(http.StatusCreated, gin.H{"message": "Rotina iniciada em segundo plano"})
}
