package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"

	"github.com/cqjcrt/blackjack/game"
	"github.com/gorilla/websocket"
)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options
var g *game.Game

type CardMessage struct {
	Type string    `json:"type"`
	Body game.Card `json:"body"`
}

type GameMessage struct {
	Type       string    `json:"type"`
	Body       game.Game `json:"body"`
	DealerHand game.Hand `json:"dealerHand"`
	Error      string    `json:"error"`
}

type JoinReply struct {
	// Type   string      `json:"type"`
	Player game.Player `json:"player"`
	SpotID string      `json:"spotId"`
}

type JoinRequest struct {
	Command string      `json:"command"`
	Player  game.Player `json:"player"`
	Wager   int         `json:"wager"`
}

type PlayRequest struct {
	Command string      `json:"command"`
	Player  game.Player `json:"player"`
}

type ErrorMessage struct {
	Type  string    `json:"type"`
	Body  game.Game `json:"body"`
	Error string    `json:"error"`
}

func main() {
	g = &game.Game{}
	g.Init()

	http.HandleFunc("/ws/hello", hello)
	http.HandleFunc("/", root)
	// fs := http.FileServer(http.Dir("static/"))
	// http.HandleFunc("/static/", http.StripPrefix("/static/", fs))
	log.Fatal(http.ListenAndServe(*addr, nil))
}

func root(w http.ResponseWriter, r *http.Request) {
	path := "." + r.URL.Path
	if path == "./" {
		path = "./static/index.html"
	}
	http.ServeFile(w, r, path)
}

func hello(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	var current *game.Spot
	for {
		// 4 phases
		// * wager loop
		// * deal loop, broadcast cards (no player interaction)
		// * player turn loop
		// * dealer turn, settle (no player interaction)

		// wager (join) loop
		log.Printf("Wager loop\n")
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			// cmd := string(message[:len(message)])
			// n := bytes.IndexByte(byteArray, 0)

			var req JoinRequest
			err = json.Unmarshal(message, &req)
			if err != nil {
				log.Printf("json: error unmarshaling %+v", message)
				continue
			}

			cmd := req.Command

			// take player input
			if cmd == "JOIN" {
				spot := g.AddPlayer(req.Player.Name)
				// TODO: for now, find a spot and set a player to it
				spot.Wager = req.Wager
				spot.Status = game.Wagered
				mp, err := json.Marshal(&JoinReply{Player: *g.Players[spot.PlayerID], SpotID: spot.ID.String()})
				if err != nil {
					log.Printf("json: %s", err)
					continue
				}
				c.WriteMessage(mt, mp)
			}
			g.Status = game.Dealing

			if g.PlayerWagered() {
				break
			}
		}

		// dealing phase (no user interaction)
		log.Printf("Dealing phase\n")
		g.Cleanup()
		g.Deal()
		// notify all players current game state
		// msg, err := json.Marshal(&GameMessage{Type: "Game", Body: *g})
		// if err != nil {
		// 	log.Printf("json: error marshaling game state %s", err)
		// }
		gmsg := &GameMessage{Type: "Game", Body: *g}
		err = c.WriteJSON(gmsg)
		if err != nil {
			log.Printf("ws: error writing message %v", gmsg)
		}

		current = &g.Spots[0]
		log.Printf("current: %+v", current)
		// player turn
		// TODO: loop over each spot (ws connection?)
		log.Printf("Player turn loop\n")
		for {
			mt, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				break
			}

			// cmd := string(message[:len(message)])
			// n := bytes.IndexByte(byteArray, 0)

			var req PlayRequest
			err = json.Unmarshal(message, &req)
			if err != nil {
				log.Printf("json: error unmarshaling %+v", message)
				continue
			}

			log.Printf("req: %+v", req)

			// if g.Status == game.PlayerTurn && current.Player.ID != req.Player.ID {
			// 	log.Printf("game: expected player %s, got %s instead", current.Player.ID, req.Player.ID)
			// 	continue
			// }
			// TODO: verify spot

			cmd := req.Command

			// take player input
			if cmd == "HIT" {
				var msg []byte
				card, err := current.Hit(g.Draw())
				if err != nil {
					log.Println("error: ", err)
					if be, ok := err.(*game.BustedError); ok {
						msg, err = json.Marshal(&GameMessage{Type: "Game", Body: *g, Error: be.Error()})
						if err != nil {
							log.Printf("json: error marshalling hit error - %s", err)
						}
						c.WriteMessage(mt, msg)
					}
					continue
				}

				msg, err = json.Marshal(&CardMessage{Type: "Card", Body: *card})
				// catch busted error
				if err != nil {
					log.Printf("json: error marshalling hit message: %s", err)
				}
				c.WriteMessage(mt, msg)
			} else if cmd == "DOUBLE" {
				current.Double(g.Draw())
			} else if cmd == "STAND" {
				current.Stand()
			}

			if g.PlayerStandOrBusted() {
				break
			}
		}

		// dealer turn
		log.Printf("Dealer turn\n")
		g.Finish()
		g.Settle()
		// notify all players current game state
		// msg, err = json.Marshal(&GameMessage{Type: "Game", Body: *g})
		// if err != nil {
		// 	log.Printf("json: error marshaling game state %s", err)
		// }
		gmsg = &GameMessage{Type: "Game", Body: *g, DealerHand: g.DealerHand()}
		c.WriteJSON(gmsg)
	}
}
