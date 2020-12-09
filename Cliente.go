package main

import (
	"fmt"
	"net"
	"encoding/gob"
	"bufio"
	"container/list"
	"io/ioutil"
	"os"
)

const conexion = ":9999"
const protocoloConexion = "tcp"
const defaultID = 0
const contID = 1
const tamanoMaxArchivo = 3000000      
const tamanoMbEnBytes = 1000000    

type Cliente struct {
	Conexion  string
	Protocolo string
	ID        int
	Login     bool
	Logout    bool
}

type Paquete struct {
	Mensaje     string
	File        Archivo
	EsMensaje   bool
	EsArchivo   bool
	IDCliente   int
	AccesoLogin bool
	Login       bool
	Logout      bool
	FinServidor bool
}

type AdminPaquetes struct {
	Paquetes             list.List
	MostrarPaquetes      bool
	CantPaquetesAnterior int
}

type Archivo struct {
	Nombre    string
	Contenido []byte
}

func (p *Paquete) esLogin() bool {
	if p.EsMensaje {
		if p.Mensaje == "Login" {
			if p.Login {
				return true
			}
		}
	}
	return false
}

func (p *Paquete) esAccesoLogin() bool {
	if p.EsMensaje {
		if p.Mensaje == "Login" {
			if p.AccesoLogin {
				return true
			}
		}
	}
	return false
}

func (p *Paquete) esLogout() bool {
	if p.EsMensaje {
		if p.Mensaje == "Logout" {
			if p.Logout {
				return true
			}
		}
	}
	return false
}

func (a *Archivo) guardarContenido(rutaArchivo string) {
	file, err := os.Open(rutaArchivo)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	contenido, err := ioutil.ReadAll(reader)
	if err != nil {
		fmt.Println(err)
		return
	}

	a.Nombre = file.Name()
	a.Contenido = contenido
}

func (c *Cliente) enviarMensaje(mensaje string, conn net.Conn, a *AdminPaquetes) {
	p := Paquete{
		Mensaje:     mensaje,
		EsMensaje:   true,
		EsArchivo:   false,
		IDCliente:   c.ID,
		AccesoLogin: false,
		Login:       false,
		Logout:      false,
		FinServidor: false,
	}
	if c.Login {
		p.AccesoLogin = true
	} else if c.Logout {
		p.Logout = true
	}
	err := gob.NewEncoder(conn).Encode(p)
	if err != nil {
		return
	}
	if !c.Login {
		a.Paquetes.PushBack(p)
	}
}

func (c *Cliente) enviarArchivo(rutaArchivo string, conn net.Conn, a *AdminPaquetes) {
	var archivo Archivo
	file, err := os.Stat(rutaArchivo)
	if err != nil {
		fmt.Println(err)
		return
	}
	if file.Size() <= tamanoMaxArchivo {
		archivo.guardarContenido(rutaArchivo)
		p := Paquete{
			File:        archivo,
			EsMensaje:   false,
			EsArchivo:   true,
			IDCliente:   c.ID,
			AccesoLogin: false,
			Login:       false,
			Logout:      false,
			FinServidor: false,
		}
		err = gob.NewEncoder(conn).Encode(p)
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("El archivo ha sido enviado")
		a.Paquetes.PushBack(p)
	} else {
		fmt.Println("El tamaño de archivo:", file.Size()/tamanoMbEnBytes, "MB, superior al límite: ", tamanoMaxArchivo/tamanoMbEnBytes, "mb")
	}
}

func (a *AdminPaquetes) existenPaquetes() bool {
	return a.Paquetes.Len() > 0
}

func leerPaquetes(conn net.Conn, a *AdminPaquetes, c *Cliente) {
	for {
		var p Paquete
		err := gob.NewDecoder(conn).Decode(&p)
		if err != nil {
			fmt.Println("El servidor se ha desconectado")
			fmt.Println("Presione enter para terminar...")
			c.Logout = true
			return
		}
		if p.esAccesoLogin() {
			c.ID = p.IDCliente
			p.AccesoLogin = false
			p.Login = true
		} else if p.terminarServidor() {
			fmt.Println("El servidor se ha desconectado")
			c.Logout = true
			return
		}
		a.Paquetes.PushBack(p)
	}
}

func mostrarPaquetes(a *AdminPaquetes, c *Cliente) {
	for {
		if a.MostrarPaquetes {
			if a.existenPaquetes() {
				if a.CantPaquetesAnterior < a.Paquetes.Len() {
					a.CantPaquetesAnterior = a.Paquetes.Len()
					fmt.Println("\n\n****** Chat ******\n")
					for e := a.Paquetes.Front(); e != nil; e = e.Next() {
						p := e.Value.(Paquete)
						if p.IDCliente == c.ID {
							fmt.Printf("%s%d(Tú)\t", "Usuario_", p.IDCliente)
						} else {
							fmt.Printf("%s%d    \t", "Usuario_", p.IDCliente)
						}
						if p.EsMensaje {
							if p.esLogin() {
								fmt.Printf("Login: Entro un usuario\n")
							} else if p.esLogout() {
								fmt.Printf("Logout: Salio un usuario\n")
							} else {
								fmt.Printf("Mensaje:    %s\n", p.Mensaje)
							}
						} else {
							fmt.Printf("Archivo:  : %s\n", p.File.Nombre)
						}
					}
					fmt.Println("\nPresione enter para salir del chat...")
				}
			}
		}
	}
}

func (p *Paquete) terminarServidor() bool {
	if p.EsMensaje {
		if p.Mensaje == "Logout" {
			if p.FinServidor {
				return true
			}
		}
	}
	return false
}

func enviarMensaje(conn net.Conn, c *Cliente, a *AdminPaquetes) {
	fmt.Println("\n\n****** Enviar mensaje de texto ******\n")
	consoleReader := bufio.NewReader(os.Stdin)
	fmt.Printf("Escriba su mensaje: ")
	mensaje, _ := consoleReader.ReadString('\n')
	mensaje = mensaje[0 : len(mensaje)-2]
	c.enviarMensaje(mensaje, conn, a)
	fmt.Println("El mensaje fue enviado")
}

func enviarArchivo(conn net.Conn, c *Cliente, a *AdminPaquetes) {
	fmt.Println("\n\n****** Enviar archivo ******\n")
	consoleReader := bufio.NewReader(os.Stdin)
	fmt.Printf("Ingrese la ruta del archivo: ")
	ruta, _ := consoleReader.ReadString('\n')
	ruta = ruta[0 : len(ruta)-2]
	if existeArchivo(ruta) {
		c.enviarArchivo(ruta, conn, a)
	} else {
		fmt.Println("No existe la ruta o el archivo ingresado")
	}
}

func existeArchivo(ruta string) bool {
	file, err := os.Open(ruta)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

func main() {
	a := new(AdminPaquetes)
	a.MostrarPaquetes = false
	a.CantPaquetesAnterior = 0
	c := new(Cliente)
	c.Conexion = conexion
	c.Protocolo = protocoloConexion
	c.ID = defaultID
	c.Logout = false

	conn, err := net.Dial(c.Protocolo, c.Conexion)
	if err != nil {
		fmt.Println("No fue posible conectarse con el servidor en", c.Conexion)
		return
	}
	
	c.Login = true
	c.enviarMensaje("Login", conn, a)
	c.Login = false
	go leerPaquetes(conn, a, c)
	go mostrarPaquetes(a, c)
	for !c.Logout {
		if c.ID != defaultID {
			var opc int
			fmt.Printf("\n\n****** Cliente: Usuario_%d ******\n", c.ID)
			fmt.Println("1) Enviar mensaje")
			fmt.Println("2) Enviar archivo")
			fmt.Println("3) Mostrar mensajes o archivos")
			fmt.Println("0) Salir")
			fmt.Printf("\nIngrese una opción: ")
			fmt.Scanln(&opc)
			switch opc {
			case 1:
				enviarMensaje(conn, c, a)
				fmt.Println("\nPresione enter para continuar...")
				fmt.Scanf("\n")
			case 2:
				enviarArchivo(conn, c, a)
				fmt.Println("\nPresione enter para continuar...")
				fmt.Scanf("\n")
			case 3:
				if a.existenPaquetes() {
					a.CantPaquetesAnterior = 0
					a.MostrarPaquetes = true
					fmt.Scanf("\n")
					a.MostrarPaquetes = false
				} else {
					fmt.Println("No existen mensajes ni archivos")
					fmt.Println("\nPresione enter para continuar...")
					fmt.Scanf("\n")
				}
			case 0:
				c.Logout = true
				c.enviarMensaje("Logout", conn, a)
				fmt.Println("\nSesión finalizada")
			default:
				fmt.Println("\nOpción no válida")
				fmt.Println("\nPresione enter para continuar...")
				fmt.Scanf("\n")
			}
		}
	}
}