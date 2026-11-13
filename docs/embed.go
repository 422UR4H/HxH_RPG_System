// Package docs embute os documentos de jogo (docs/game) no binário.
package docs

import "embed"

// GameFS contém os docs de jogo voltados ao jogador. Os caminhos são
// relativos a esta pasta, ex.: "game/classes.md".
//
//go:embed game/*.md game/ficha-de-personagem/*.md
var GameFS embed.FS
