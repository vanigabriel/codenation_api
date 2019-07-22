package main

import (
	"bufio"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	//_ "./docs/docs.go" // For gin-swagger

	"github.com/gen2brain/go-unarr"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	_ "github.com/lib/pq"
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

// FuncPublico guarda o funcionario antes de importar para o banco
type FuncPublico struct {
	Nome      string
	Cargo     string
	VlSalario float64
}

func setupRouter() *gin.Engine {
	// Disable Console Color
	// gin.DisableConsoleColor()
	r := gin.Default()

	r.GET("/welcome", func(c *gin.Context) {
		firstname := c.DefaultQuery("firstname", "Guest")
		lastname := c.Query("lastname") // shortcut for c.Request.URL.Query().Get("lastname")

		c.String(http.StatusOK, "Hello %s %s", firstname, lastname)
	})

	r.POST("/form_post", func(c *gin.Context) {
		message := c.PostForm("message")
		nick := c.DefaultPostForm("nick", "anonymous")

		c.JSON(200, gin.H{
			"status":  "posted",
			"message": message,
			"nick":    nick,
		})
	})

	r.POST("/login", Login)

	authorized := r.Group("/", gin.BasicAuth(gin.Accounts{
		"admin": "admin",
	}))

	authorized.GET("dale", dale)
	authorized.POST("users", registerUser)
	// Get Users
	// Get notificações
	// Get Clientes
	// Funcionarios publicos dos ultimos meses
	// Post Cliends upload
	// Verificar Docker
	// DAshboard : Qtd total de +20mil, qtd q eu detenho, valor, dos ultimos meses, o mairo valor de salário, menor valor de salário, média, qtd de pessoa por orgão
	authorized.POST("admin", registerAdministrator)

	return r
}

func main() {

	handleError(godotenv.Load()) //Load environmenatal variables

	r := setupRouter()
	// Listen and Server in 0.0.0.0:8080
	r.Run(":8080")

}

func dale(c *gin.Context) {
	c.String(http.StatusOK, "regatao")
}

func baixarCSV() error {
	log.Println("Iniciando BaixarCSV")
	filepath := "remuneracao.rar"
	actualPath, _ := os.Getwd()

	// Monta a requisição para baixar o CSV
	body := strings.NewReader(`__EVENTTARGET=&__EVENTARGUMENT=&__LASTFOCUS=&__VIEWSTATE=%2FwEPDwULLTIwNDQzOTAyMzEPZBYCAgMPZBYMAgUPEA8WBh4ORGF0YVZhbHVlRmllbGQFCE9SR0FPX0lEHg1EYXRhVGV4dEZpZWxkBQpPUkdBT19ERVNDHgtfIURhdGFCb3VuZGdkEBVbBVRPRE9THUFETUlOSVNUUkFDQU8gR0VSQUwgRE8gRVNUQURPHUFHLk0uVi5QQVIuTElULk5PUlRFIEFHRU1WQUxFHEFHLlJFRy5TQU4uRU4uRVNULlNQLiBBUlNFU1AeQUcuUkVHLlNWLlAuREVMLlRSLkUuU1AgQVJURVNQHUFHRU5DSUEgTUVULkNBTVBJTkFTIEFHRU1DQU1QHkFHRU5DSUEgTUVUUk9QLkIuU0FOVElTVEEgQUdFTShBR0VOQ0lBIE1FVFJPUE9MSVRBTkEgREUgQ0FNUElOQVMgLSBBR0VNKEFHRU5DSUEgTUVUUk9QT0xJVEFOQSBERSBTT1JPQ0FCQSAtIEFHRU0eQy5FLkVELlRFQy5QQVVMQSBTT1VaQS1DRUVURVBTHUMuUC5UUkVOUyBNRVRST1BPTElUQU5PUy1DUFRNHUNBSVhBIEJFTkVGSUMuUE9MSUNJQSBNSUxJVEFSCkNBU0EgQ0lWSUwdQ0VURVNCLUNJQS5BTUJJRU5UQUwgRVNULlMuUC4aQ0lBIERFU0VOVi5BR1JJQy5TUCBDT0RBU1AdQ0lBLkRFUy5IQUIuVVJCLkVTVC5TLlAuLUNESFUeQ0lBLlBBVUxJUy5TRUNVUklUSVpBQ0FPLUNQU0VDHkNJQS5QUk9DLkRBRE9TIEVTVC5TLlAtUFJPREVTUChDSUEuU0FORUFNRU5UTyBCQVNJQ08gRVNULlMuUEFVTE8tU0FCRVNQHkNJQS5TRUdVUk9TIEVTVC5TLlBBVUxPLUNPU0VTUB1DT01QLk1FVFJPUE9MSVRBTk8gUy5QLi1NRVRSTx1DT01QQU5ISUEgRE9DQVMgU0FPIFNFQkFTVElBTyhDT01QQU5ISUEgUEFVTElTVEEgREUgT0JSQVMgRSBTRVJWSUNPUyAtBURBRVNQHURFUEFSVEFNLkVTVFJBREFTIFJPREFHRU0gREVSKERFUEFSVEFNRU5UTyBBR1VBUyBFTkVSR0lBIEVMRVRSSUNBLURBRUUoREVQQVJUQU1FTlRPIEVTVEFEVUFMIERFIFRSQU5TSVRPLURFVFJBTh5ERVBUTy4gRVNULiBUUkFOU0lUTyBERVRSQU4gU1AeREVTRU5WT0xWLlJPRE9WSUFSSU8gUy9BLURFUlNBKERFU0VOVk9MVkUgU1AgQUdFTkNJQSBERSBGT01FTlRPIERPIEVTVEEoRU1BRS1FTVBSRVNBIE1FVFJPUE9MSVRBTkEgREUgQUdVQVMgRSBFTh5FTVAuTUVUUi5UUi5VUkIuU1AuUy9BLUVNVFUtU1AoRU1QLlBBVUxJU1RBIFBMQU5FSi5NRVRST1BMSVRBTk8gUy5BLUVNUBpGQUMuTUVELlMuSi5SLlBSRVRPLUZBTUVSUBtGQUMuTUVESUNJTkEgTUFSSUxJQS1GQU1FTUEaRklURVNQLUpPU0UgR09NRVMgREEgU0lMVkEeRlVORC5BTVBBUk8gUEVTUS5FU1QuU1AtRkFQRVNQHkZVTkQuQ09OUy5QUk9ELkZMT1JFU1RBTCBFLlNQLhxGVU5ELk1FTU9SSUFMIEFNRVJJQ0EgTEFUSU5BHUZVTkQuUEFSUVVFIFpPT0xPR0lDTyBTLlBBVUxPHkZVTkQuUEUuQU5DSElFVEEtQy5QLlJBRElPIFRWLh1GVU5ELlBGLkRSLk0uUC5QSU1FTlRFTC1GVU5BUB5GVU5ELlBSRVYuQ09NUEwuRVNULlNQIFBSRVZDT00eRlVORC5QUk8tU0FOR1VFLUhFTU9DRU5UUk8gUy5QHEZVTkQuUkVNLlBPUC4gQy5ULkxJTUEgLUZVUlAeRlVORC5TSVNULkVTVC5BTkFMLkRBRE9TLVNFQURFHkZVTkQuVU4uVklSVFVBTCBFU1QuU1AgVU5JVkVTUBFGVU5EQUNBTyBDQVNBLVNQLhxGVU5EQUNBTyBERVNFTlYuRURVQ0FDQU8tRkRFHUZVTkRBQ0FPIE9OQ09DRU5UUk8gU0FPIFBBVUxPD0ZVTkRBQ0FPIFBST0NPThZHQUJJTkVURSBETyBHT1ZFUk5BRE9SGkguQy5GQUMuTUVELkJPVFVDQVRVLUhDRk1CHUhDIEZBQyBNRURJQ0lOQSBSSUIgUFJFVE8gVVNQGUhPU1AuQ0xJTi5GQUMuTUVELk1BUklMSUEdSE9TUC5DTElOLkZBQy5NRUQuVVNQLUhDRk1VU1AeSU1QUi5PRklDSUFMIEVTVEFETyBTLkEuIElNRVNQHklOU1QgTUVEIFNPQyBDUklNSU5PIFNQLSBJTUVTQx5JTlNULkFTLk1FRC5TRVJWLlAuRVNULiBJQU1TUEUeSU5TVC5QQUdUT1MuRVNQRUNJQUlTIFNQLUlQRVNQHklOU1QuUEVTT1MgTUVESUQuRS5TLlAtSVBFTS9TUB5JTlNULlBFU1EuVEVDTk9MT0dJQ0FTIEVTVC5TLlAdSlVOVEEgQ09NRVJDLkUuUy5QQVVMTy1KVUNFU1AeUEFVTElTVFVSIFNBLkVNUFIuVFVSLkVTVC5TLlAuGVBPTElDSUEgTUlMSVRBUiBTQU8gUEFVTE8cUFJPQ1VSQURPUklBIEdFUkFMIERPIEVTVEFETx5TQU8gUEFVTE8gUFJFVklERU5DSUEgLSBTUFBSRVYeU0VDLlRSQU5TUE9SVEVTIE1FVFJPUE9MSVRBTk9THlNFQ1IuQUdSSUNVTFRVUkEgQUJBU1RFQ0lNRU5UTx5TRUNSLkNVTFRVUkEgRUNPTk9NSUEgQ1JJQVRJVkEeU0VDUi5ERVNFTlZPTFZJTUVOVE8gRUNPTk9NSUNPHlNFQ1IuRVNULkRJUi5QRVMuQy9ERUZJQ0lFTkNJQR5TRUNSRVQgREUgUkVMQUNPRVMgRE8gVFJBQkFMSE8eU0VDUkVULkFETUlOSVNUUi5QRU5JVEVOQ0lBUklBHlNFQ1JFVC5TQU5FQU1FTlRPIFJFQy5ISURSSUNPUx1TRUNSRVRBUi5GQVpFTkRBIFBMQU5FSkFNRU5UTxZTRUNSRVRBUklBIERBIEVEVUNBQ0FPF1NFQ1JFVEFSSUEgREEgSEFCSVRBQ0FPE1NFQ1JFVEFSSUEgREEgU0FVREUdU0VDUkVUQVJJQSBERSBERVNFTlZPTFZJTUVOVE8WU0VDUkVUQVJJQSBERSBFU1BPUlRFUxVTRUNSRVRBUklBIERFIEdPVkVSTk8eU0VDUkVUQVJJQSBERSBMT0dJU1RJQ0EgRSBUUkFOFVNFQ1JFVEFSSUEgREUgVFVSSVNNTxtTRUNSRVRBUklBIERFU0VOVi4gUkVHSU9OQUweU0VDUkVUQVJJQSBFTkVSR0lBIEUgTUlORVJBQ0FPHVNFQ1JFVEFSSUEgSU5GLiBNRUlPIEFNQklFTlRFHlNFQ1JFVEFSSUEgSlVTVElDQSBFIENJREFEQU5JQRxTRUNSRVRBUklBIFNFR1VSQU5DQSBQVUJMSUNBHVNVUEVSSU5ULkNPTlRSLkVOREVNSUFTLVNVQ0VOKFNVUEVSSU5URU5ERU5DSUEgREUgQ09OVFJPTEUgREUgRU5ERU1JQVMVWwItMQExATIBMwE0ATUBNgE3ATgBOQIxMAIxMQIxMgIxMwIxNAIxNQIxNgIxNwIxOAIxOQIyMAIyMQIyMgIyMwIyNAIyNQIyNgIyNwIyOAIyOQIzMAIzMQIzMgIzMwIzNAIzNQIzNgIzNwIzOAIzOQI0MAI0MQI0MgI0MwI0NAI0NQI0NgI0NwI0OAI0OQI1MAI1MQI1MgI1MwI1NAI1NQI1NgI1NwI1OAI1OQI2MAI2MQI2MgI2MwI2NAI2NQI2NgI2NwI2OAI2OQI3MAI3MQI3MgI3MwI3NAI3NQI3NgI3NwI3OAI3OQI4MAI4MQI4MgI4MwI4NAI4NQI4NgI4NwI4OAI4OQI5MBQrA1tnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnZ2dnFgFmZAIHDxBkEBUBBVRPRE9TFQECLTEUKwMBZxYBZmQCCQ8QDxYGHwAFC1NJVFVBQ0FPX0lEHwEFDVNJVFVBQ0FPX0RFU0MfAmdkEBUEBVRPRE9TC0FQT1NFTlRBRE9TBkFUSVZPUwxQRU5TSU9OSVNUQVMVBAItMQExATIBMxQrAwRnZ2dnFgFmZAILDw9kFgIeCm9uS2V5UHJlc3MFJ3JldHVybiBNYXNjYXJhTW9lZGEodGhpcywnLicsJywnLGV2ZW50KWQCDQ8PZBYCHwMFJ3JldHVybiBNYXNjYXJhTW9lZGEodGhpcywnLicsJywnLGV2ZW50KWQCFQ9kFgJmD2QWBAIBDxYCHgdWaXNpYmxlaGQCAw8PFgIfBGhkFgICAw88KwARAgEQFgAWABYADBQrAABkGAIFHl9fQ29udHJvbHNSZXF1aXJlUG9zdEJhY2tLZXlfXxYBBQxpbWdFeHBvcnRUeHQFBGdyaWQPZ2RQPxWdkKW2N7k2sc3cRkrsKaN6oX%2FYV5km4LhQ5LBTcA%3D%3D&__VIEWSTATEGENERATOR=E42B1F40&__EVENTVALIDATION=%2FwEdAG7zLJ4CWjEZheF5kVSEbhUBha8fMqpdVfgdiIcywQp19AS0oC9%2BkRn5wokBQj%2BYmSdj%2FRE4%2FVY2xVooDbyNylWSFXsupcqZ9EYohXUHrvyuvszqcPgWZLCNPbx1As5K6XI8YfiXwzc6jdd6doCEWNMhfUq2YkY3rbVwieJI30sGRBiYwU43rbtypsxax6Lexvr9tn%2FppXosAOoaLiPglbLZDQ4AHCggkRiV1y9R5Jk3hxzIBiDVeBd4ex%2FDPERS7Y3hxS83fVJEzO6I%2BsKPdRPTZbKZKzZ%2FiI%2Fo2LERffiPWbY0qpjFHBt23vPUuehVkAOA1ngNB93rbK%2Bu0E54XcLAmWLN%2Fl%2Bz5m0ApRDNS4L3FwTfILDr1aT4Crd1%2F2X2tGTSlHv5v4gI%2B%2F4UxQdVOOXcJIWT3hhEHPLkfTczdhS%2BJPFzCLQyhLlM%2FTIkVLdCEWiXz8XDG1%2BqV0wHjm1sFCkHt5aLy6yjxTyv1FFML9B%2Fo0JBJO%2By%2B74vfDQlvwQWQHtswD%2Bjri2Ja0FbYTVaHetzL3nIpMtKnzHrJejZWNnngPadPS2744kvbqzTJQaAdqOeYy%2FXyO581zGaQB16a5HkpT5jddxT22MOtOJS9%2BOuUHRXp8dj268DwFDqeWohT0vm1b0FOlCVjyi8V9MKHPYPpHgZ%2F2GzcT5zaEXX3Wa7dGMCaXmo3KMrfSTIEMtzpixzPEyfillVBjlMq8fiaJmavKW63uZc65AHMJEgzJBWOOnY33pftn93IOwZzZWV8DBA7v%2F9aPpqFJWx65SrmQqSjTKR9Q8znWzwmOcZE4%2FSuTP7i%2BXb7NoOWr4anBMJ9L8iQIpPyUdRVhTh0dqpW9mg677VkTJzeFDr78YgZsAwP%2FX%2BdTV%2FINjSEi5I3GKGi7myZ7%2BjeKd7PDtAjn8O4hLTJfL4LFg4Nvwdmd%2F53R8Jw4b9e%2FlLobx4zXIq3GAuywAjOQvHY8AEnfNd%2FlXdKYxyzc%2FwfpCNJupjNVpUse2VJD4oS1BuBPCBdQ5aaErF4JFlItPtLQCYFzs0jfHra3vGXa5DUmVxUHX61STePVHIx%2Bb2IzWzaVJbMWnr0ySeyyy%2FZ1AEi%2FGyAY4VRi7gupaG4KIpRnL0PqiHkB0m%2BFOAGOzlYyAzkRO1hwDnOQf3fkyzTk8GPsW4ORs6zPd%2BeDosaOUhW1MEtWA%2BSqsohtmqkoKbjumKVbQvus3TM3adBbzpeRPEjnLNywu7OwRAhFtyU0gmtXU9am1kuUbvzTaW93G%2FXW5pJhxIEGLJ46ijUCocW5ypp1AUfwUVaLtxxktia9eKFUCg16rKs9CfE8mQS1sJL8sXrl1kCYgl357rWaG95jfZ509s%2Bm2fA%2BOt0aP8OyaOU4R1ht8FAaoUaukJi9ac%2B52YAhiIATqgCuAVAUaz6iVZ30v9i3l79pG%2FQjT0yzItrPhgpeaj5FDDRNwFWQfE5v7dhuWXa0fqNuT0%2F3rHd8yAI%2FR31smXtVMpuDg4uNPHIl%2B2FxKOozxg%2Fv%2B%2BE9d%2FZoPPgEhC0wqwEcy5cuqQMsS7I2iwe1Xfp9TBV2uBNFpR3V1ws1NcSb0O892YPaDPsxrja2GQM7SzAShZDNlCOSW7Tt%2Fu0g%2BeirEQ%2FlwLvd%2FyO3h%2FPXkp4oZAfoeCSWuKxs7UkSXX7piPjdZRkxS8%2B1Tv52TtsW%2F%2FarETeAIdqgWD21SCG%2F%2BSG%2FyFJtRwUalOOSCKwgXmjHLagrrOpyOVvrzcda9t4I8AvfZJNBX4HCyHl%2F8v7zlaXsN6v3xdx7SBYcgTu1GewkDpUJSUGbiJpTFb9FwFesoo5ATV8LN38tAuINPU8rfSikTUmdlp8CARYKFn95WsBdjs1x8c6lK59jnQ%2FQHi2nKDMKfdQRVhcvnFwvt6SokCFQDX7AEtmU9OC%2Fkwe5SIcBU04jVZdwLiKogB2pPql%2FnA4CHA7mEf3AIr0wLOnRAQ0xjhC3PXHrIjjpV2suu3zMJ7LscXSxIToHr95TxJTzSEj9C7XyN%2FGMISH%2FTKb%2FPRxrbwGTEZF3x922wvTvFKuuxNUJFB79U3ZPxLws5iIazIlee0zV3InWYYPP26JIa5R0Em8ORb%2B%2FoUDlJKcdv6NoWV%2F5WtCyREa2Rxke5ZukLmT7xiWinv8jrwbnAz1AUaMm8xKsc4G6dNWu2jHrgAaNFlmOLZIeG0OTsyPhh%2B%2F0WQdOTAD9zAblcx6VvMEe43r2g9sGn75bO7ZW6nZ7hGBjKUqSH4S7Qy5ngR%2FiduIfdzD0oNgNO6zlZmgx%2BPVHfpxvG%2B1lXBZBLAe6JyY9%2FwY3j6%2BMGuruxn5MX0jsPeyBXK401Kwjl8g4KbJ6y3JnlYwpVFE%2BxaAvUaNHQI16ZHBEZs26yaBXQzbLC2jFI6XXFnHVbAsVbJ&txtNome=&orgao=-1&cargo=-1&situacao=-1&txtDe=&txtAte=&hdInicio=&hdFinal=&hdPaginaAtual=&hdTotal=&imgExportTxt.x=15&imgExportTxt.y=22`)
	req, err := http.NewRequest("POST", "http://www.transparencia.sp.gov.br/PortalTransparencia-Report/Remuneracao.aspx", body)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:67.0) Gecko/20100101 Firefox/67.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Referer", "http://www.transparencia.sp.gov.br/PortalTransparencia-Report/Remuneracao.aspx")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Cookie", "ASP.NET_SessionId=jq2ser3k5kslyjkore4fvplw")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	log.Println("Iniciando download")
	//Faz a requisição
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Println("Download finalizado")

	// Cria arquivo
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	log.Println("Salvando localmente")
	// Escreve o CSV no arquivo
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	// Descompacta
	newFile, err := Unrar(filepath, actualPath)
	if err != nil {
		return err
	}

	// Importa para o banco
	err = importCSV(newFile)
	if err != nil {
		return err
	}

	// Remove arquivo
	err = os.Remove(newFile)
	if err != nil {
		return err
	}

	log.Println("Finalizando BaixarCSV")
	return err
}

// Unrar will decompress a zip archive, moving all files and folders
// within the rar file (parameter 1) to an output directory (parameter 2).
func Unrar(src string, dest string) (string, error) {
	log.Println("Descompactando arquivo " + src)
	r, err := unarr.NewArchive(src)
	if err != nil {
		return "", err
	}
	defer r.Close()

	err = r.Extract(dest)
	if err != nil {
		panic(err)
	}

	err = r.Extract("")
	if err != nil {
		panic(err)
	}

	err = os.Rename("remuneracao.txt", "remuneracao.csv")

	return "remuneracao.csv", err
}

func importCSV(src string) error {

	log.Println("Abrindo conexão com o banco para importar")
	db, err := initDB()
	if err != nil {
		return err
	}
	defer db.Close()

	log.Println("Limpando tabela funcionarios_publicos")

	// Limpa a tabela
	sql := "delete from funcionarios_publicos"
	_, err = db.Exec(sql)
	if err != nil {
		return err
	}

	// Abrindo arquivo
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	log.Println("Iniciando leitura do arquivo")
	// Inicia leitura do CSV
	r := csv.NewReader(bufio.NewReader(f))

	r.Read()      //retira cabeçalho
	r.Comma = ';' //altera separador

	// apenas alfa-numericos e espaços
	reg, err := regexp.Compile("[^a-zA-Z0-9 ]+")
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Iniciando transação")
	// Abre transação
	trc, _ := db.Begin()

	// Iteração pelo arquivo
	for {
		record, err := r.Read()

		if err == io.EOF {
			break
		}

		// Ajustando o separador de valor
		s := strings.Replace(record[3], ",", ".", -1)
		salario, err := strconv.ParseFloat(s, 64) //converte pra float

		// Objeto temporario
		fTemp := FuncPublico{
			Nome:      reg.ReplaceAllString(record[0], ""),
			Cargo:     reg.ReplaceAllString(record[1], ""),
			VlSalario: salario}

		// Insert na tabela, se tiver conflito não faz nada
		sql := "insert into funcionarios_publicos as v (nm_funcionarios, nm_cargo, vl_salario) values ($1, $2, $3) ON CONFLICT ON CONSTRAINT funcionarios_publicos_pkey DO nothing"
		_, err = trc.Exec(sql, fTemp.Nome, fTemp.Cargo, fTemp.VlSalario)
		if err != nil {
			trc.Rollback()
			return err
		}

	}
	err = trc.Commit()
	log.Println("Commit efetuado")

	return err
}

func handleError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type Credentials struct {
	Password string `json:"password"`
	Username string `json:"username"`
}
type Users struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func registerUser(c *gin.Context) {
	usrs := &Users{}
	c.BindJSON(&usrs)

	if usrs.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados obrigatórios não recebidos"})
	}

	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível conectar ao banco de dados"})
	}
	defer db.Close()

	if rowExists("SELECT id FROM usuarios WHERE email=$1", db, usrs.Email) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email já cadastrado"})
	}

	_, err = db.Exec("INSERT INTO usuarios (name, email, created_on) VALUES ($1, $2, now())", usrs.Name, usrs.Email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível conectar ao banco de dados"})
	}
}

func Login(c *gin.Context) {
	creds := &Credentials{}
	c.BindJSON(&creds)

	if creds.Username == "" || creds.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dados obrigatórios não recebidos"})
	}

	db, err := initDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível conectar ao banco de dados"})
	}
	defer db.Close()

	row := db.QueryRow("SELECT password FROM administradores WHERE username=$1", creds.Username)

	storedCreds := &Credentials{}
	err = row.Scan(&storedCreds.Password) // guardando a passw para comparar
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "usuário não encontrado."})
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Não foi possível conectar ao banco de dados"})
	}

	//err = bcrypt.CompareHashAndPassword([]byte(storedCreds.Password), []byte(creds.Password));

	if storedCreds.Password != creds.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Senha incorreta"})
		return
	}

	//se passou, passamos o usuario e senha
	c.JSON(http.StatusOK, gin.H{"user": "admin", "pass": "admin"})
}

func rowExists(query string, db *sql.DB, args ...interface{}) bool {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := db.QueryRow(query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		fmt.Printf("ERRO checando se existe '%s' %v", args, err)
	}
	return exists
}

func registerAdministrator(c *gin.Context) {
	creds := &Credentials{}
	c.BindJSON(&creds)

	hashpwd, err := bcrypt.GenerateFromPassword([]byte(creds.Password), hashCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Problemas para criptografar a senha."})
	}

	db, err := initDB()
	_, err = db.Exec("INSERT INTO administradores (username, password, created_on) VALUES ($1, $2, now())", creds.Username, string(hashpwd))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
	}
}
