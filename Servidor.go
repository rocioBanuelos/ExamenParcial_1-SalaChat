package main

import (
	"container/list"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"
)

const conexion = ":9999"
const protocoloConexion = "tcp"
const defaultID = 0
const contID = 1

type Servidor struct {
	Conexion             string
	Protocolo            string
	Paquetes             list.List
	Clientes             list.List
	Mensajes             []Paquete
	Archivos             []Paquete
	Mostrar              bool
	contID               int
	Terminar             bool
	CantPaquetesAnterior int
}

type Archivo struct {
	Nombre    string
	Contenido []byte
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

type Cliente struct {
	ID   int
	Conn net.Conn
}

func (serv *Servidor) existenMensajes() bool {
	serv.obtenerMensajes()
	return len(serv.Mensajes) > 0
}

func (serv *Servidor) obtenerMensajes() {
	serv.Mensajes = make([]Paquete, 0)
	for e := serv.Paquetes.Front(); e != nil; e = e.Next() {
		p := e.Value.(Paquete)
		if !p.FinServidor {
			serv.Mensajes = append(serv.Mensajes, p)
		}
	}
}

func (serv *Servidor) existenArchivos() bool {
	serv.obtenerArchivos()
	return len(serv.Archivos) > 0
}

func (serv *Servidor) obtenerArchivos() {
	serv.Archivos = make([]Paquete, 0)
	for e := serv.Paquetes.Front(); e != nil; e = e.Next() {
		p := e.Value.(Paquete)
		if p.EsArchivo {
			serv.Archivos = append(serv.Archivos, p)
		}
	}
}

func (serv *Servidor) mostrarArchivos() {
	serv.obtenerArchivos()
	fmt.Println("\n\n****** Lista de Archivos ******\n")
	fmt.Println("Número ")
	for i := 0; i < len(serv.Archivos); i++ {
		a := serv.Archivos[i]
		fmt.Printf("%d)\tUsuario_%d: %s\n", i+1, a.IDCliente, a.File.Nombre)
	}
}

func (serv *Servidor) existenPaquetes() bool {
	return serv.Paquetes.Len() > 0
}

func (serv *Servidor) existenPaquetesParaRespaldar() bool {
	return serv.existenMensajes() || serv.existenArchivos()
}

func (serv *Servidor) enviarPaqueteAClientes(p Paquete) {
	for e := serv.Clientes.Front(); e != nil; e = e.Next() {
		c := e.Value.(*Cliente)
		if c.ID != p.IDCliente {
			err := gob.NewEncoder(c.Conn).Encode(p)
			if err != nil {
				fmt.Println("Error: ", err)
				return
			}
		}
	}
}

func (serv *Servidor) existenClientes() bool {
	return serv.Clientes.Len() > 0
}

func (serv *Servidor) eliminarCliente(id int) {
	for e := serv.Clientes.Front(); e != nil; e = e.Next() {
		c := e.Value.(*Cliente)
		if c.ID == id {
			serv.Clientes.Remove(e)
			break
		}
	}
}

func (serv *Servidor) respaldarMensajes() {
	var mensaje string
	serv.obtenerMensajes()
	t := time.Now()
	fecha := fmt.Sprintf("%d-%02d-%02d_%02d_%02d_%02d", t.Day(), t.Month(), t.Year(), t.Hour(), t.Minute(), t.Second())
	nombreArchivo := "Respaldo-Mensajes-" + fecha + ".txt"
	file, err := os.Create(nombreArchivo)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()
	for i := 0; i < len(serv.Mensajes); i++ {
		m := serv.Mensajes[i]
		mensaje = "Usuario_" + strconv.Itoa(m.IDCliente) + "\t"
		if m.EsMensaje {
			if m.EsMensaje {
				if m.esLogin() {
					mensaje = mensaje + "Login" + ": " + "Entro el usuario" + "\n"
				} else if m.esLogout() {
					mensaje = mensaje + "Logout" + ": " + "Salio el usuario" + "\n"
				} else {
					mensaje = mensaje + "Mensaje:   " + m.Mensaje + "\n"
				}
			}
		} else {
			mensaje = mensaje + "Archivo:  " + m.File.Nombre + "\n"
		}
		file.WriteString(mensaje)
	}
	fmt.Println("El archivo de respaldo con mensajes ha sido creado:", nombreArchivo)
}

func (serv *Servidor) respaldarArchivo(numArchivo int) {
	numArchivo--
	if numArchivo >= 0 && numArchivo < len(serv.Archivos) {
		a := serv.Archivos[numArchivo]
		contenido := a.File.Contenido
		file, err := os.Create("Respaldo-" + "Archivo-ID" + strconv.Itoa(numArchivo+1)+".txt")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		if _, err := file.Write(contenido); err != nil {
			fmt.Println(err)
			return
		}
		if err := file.Sync(); err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println("Respaldo de archivo creado:", "Respaldo-"+a.File.Nombre)
	} else {
		fmt.Println("No existe el número del archivo ingresado")
	}
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

func (c *Cliente) activar(serv *Servidor) {
	go c.leer(serv)
}

func (c *Cliente) leer(serv *Servidor) {
	for {
		var p Paquete
		err := gob.NewDecoder(c.Conn).Decode(&p)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		if serv.Terminar {
			p.EsMensaje = true
			p.Mensaje = "Logout"
			p.FinServidor = true
			err := gob.NewEncoder(c.Conn).Encode(p)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			return
		}

		if p.esAccesoLogin() {
			p.IDCliente = serv.contID
			serv.contID++
			err := gob.NewEncoder(c.Conn).Encode(p)
			if err != nil {
				fmt.Println("Error:", err)
				return
			}
			p.AccesoLogin = false
			p.Login = true
			serv.Paquetes.PushBack(p)
			serv.enviarPaqueteAClientes(p)
			c.ID = p.IDCliente
			serv.Clientes.PushBack(c)
		} else if p.esLogout() {
			c.Conn.Close()
			serv.eliminarCliente(p.IDCliente)
			serv.Paquetes.PushBack(p)
			serv.enviarPaqueteAClientes(p)
			return
		} else {
			serv.Paquetes.PushBack(p)
			serv.enviarPaqueteAClientes(p)
		}
	}
}

func mostrarPaquetes(serv *Servidor) {
	for {
		if serv.Mostrar {
			if serv.CantPaquetesAnterior < serv.Paquetes.Len() {
				serv.CantPaquetesAnterior = serv.Paquetes.Len()
				fmt.Println("\n\n****** Chat ******\n")
				for e := serv.Paquetes.Front(); e != nil; e = e.Next() {
					p := e.Value.(Paquete)
					fmt.Printf("Usuario_%d    \t", p.IDCliente)
					if p.EsMensaje {
						if p.esLogin() {
							fmt.Printf("Login: Entro el usuario\n")
						} else if p.esLogout() {
							fmt.Printf("Logout: Salio el usuario\n")
						} else {
							fmt.Printf("Mensaje: %s\n", p.Mensaje)
						}
					} else {
						fmt.Printf("Archivo: %s\n", p.File.Nombre)
					}
				}
				fmt.Println("\nPresione enter para salir del chat...")
			}
		}
	}
}

func ejecutarServidor(s net.Listener, serv *Servidor) {
	for {
		c, err := s.Accept()
		if err != nil {
			continue
		}
		handleCliente(c, serv)
	}
}

func handleCliente(conn net.Conn, serv *Servidor) {
	c := new(Cliente)
	c.Conn = conn
	c.activar(serv)
}

func respaldarMensajesArchivos(serv *Servidor) {
	var num int
	var opc int
	fmt.Println("\n\n****** Respaldar mensajes o archivos enviados ******")
	fmt.Println("1) Respaldar mensajes")
	fmt.Println("2) Respaldar archivo")
	fmt.Println("0) Regresar")
	fmt.Printf("\nIngresa una opción: ")
	fmt.Scan(&opc)
	switch opc {
	case 1:
		if serv.existenMensajes() {
			serv.respaldarMensajes()
		} else {
			fmt.Println("No existen mensajes para respaldar")
		}
	case 2:
		if serv.existenArchivos() {
			serv.mostrarArchivos()
			fmt.Printf("\nNúmero de archivo para hacer un respaldo: ")
			fmt.Scan(&num)
			serv.respaldarArchivo(num)
		} else {
			fmt.Println("No existen archivos para respaldar")
		}
	default:
	}
}

func main() {
	continuar := true
	mostrar := false
	serv := new(Servidor)
	serv.CantPaquetesAnterior = 0
	serv.Conexion = conexion
	serv.Protocolo = protocoloConexion
	serv.Mostrar = false
	serv.contID = contID
	serv.Terminar = false
	s, err := net.Listen(serv.Protocolo, serv.Conexion)
	if err != nil {
		return
	}
	defer s.Close()
	go ejecutarServidor(s, serv)
	go mostrarPaquetes(serv)
	for continuar {
		var opc int
		fmt.Println("\n\n****** Servidor ******")
		fmt.Println("1) Mostrar mensajes o archivos")
		fmt.Println("2) Respaldar mensajes o archivos")
		fmt.Println("0) Salir")
		fmt.Printf("\nIngresa una opción: ")
		fmt.Scan(&opc)
		switch opc {
		case 1:
			if serv.existenPaquetes() {
				serv.CantPaquetesAnterior = 0
				serv.Mostrar = true
				fmt.Scanf("\n\n")
				serv.Mostrar = false
				mostrar = true
			} else {
				fmt.Println("\n\nNo existen mensajes ni archivos")
			}
		case 2:
			if serv.existenPaquetesParaRespaldar() {
				respaldarMensajesArchivos(serv)
			} else {
				fmt.Println("\n\nNo existen mensajes ni archivos para respaldar")
			}
		case 0:
			serv.Terminar = true
			fmt.Println("\nTerminar servidor")
			continuar = false
		default:
			fmt.Println("\nOpción no válida")
		}
		if continuar {
			if !mostrar {
				fmt.Println("\nPresione enter para continuar...")
				fmt.Scanf("\n\n")
			} else {
				mostrar = false
			}
		}
	}
}
