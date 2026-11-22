package matchsession

import (
	"errors"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/errtext"
)

var (
	ErrParticipantNotFound     = errtext.Sentence("Você não participa desta partida.")
	ErrActionActorMismatch     = errtext.Sentence("Essa ação não é sua.")
	ErrRoundHasOpenTurn        = errtext.Sentence("Feche o turno aberto antes de encerrar o round.")
	ErrCharSheetNotFound       = errtext.Sentence("Esse personagem não está na partida.")
	ErrNoActiveTurn            = errtext.Sentence("Não há turno ativo neste round.")
	ErrCharacterStatusNotFound = errtext.Sentence("Esse personagem não está na partida.")
	// ErrReactionActorMismatch means the caller tried to react through a character that is
	// not theirs. The same rule EnqueueAction enforces, on the other route.
	ErrReactionActorMismatch = errtext.Sentence("Esse personagem não é seu.")
	// ErrReactorNotTargeted means the caller was not aimed at. Bystanders watch.
	ErrReactorNotTargeted = errtext.Sentence("Só quem é alvo da ação aberta pode reagir a ela.")
	// ErrReactorAlreadyReacted means this character already has a reaction attached to the open
	// action. One reaction per character per action: the chain counts a character once
	// (buildChainOrder), so a second one would be charged and then ignored.
	ErrReactorAlreadyReacted = errtext.Sentence("Esse personagem já reagiu à ação aberta.")
	// ErrTurnAlreadyClosed means the turn the caller tried to open a reaction on has already
	// finished — there is no one left to narrate for.
	ErrTurnAlreadyClosed = errtext.Sentence("O turno já fechou, então não dá para reagir.")
	// ErrReactionNotFound means the given id is not attached to the open turn.
	ErrReactionNotFound = errtext.Sentence("Essa reação não está no turno atual.")
	// ErrNoOpenTurn is close_turn with nothing under the baton. It is an error rather than a
	// no-op because the master pressed a button that describes an action they believe is
	// happening; answering silently would leave them believing it happened.
	ErrNoOpenTurn = errtext.Sentence("Não há turno aberto para fechar.")
	// ErrActionNotOnTurn means a condition edit named an ActionID that is neither the open
	// turn's own action nor one of its attached reactions.
	ErrActionNotOnTurn = errtext.Sentence("Essa ação não está no turno aberto.")
	// ErrConditionTargetMissing means a condition edit named a Field or SkillName that names
	// nothing present on the action — a client bug and never a no-op: the master pressed a
	// control describing a test they believe exists, and answering silently would leave them
	// believing they changed it.
	ErrConditionTargetMissing = errtext.Sentence("Esse teste não existe nessa ação.")
	// ErrAmbiguousConditionEdit means a condition edit set both Field and SkillName. They are
	// alternatives, never both at once.
	ErrAmbiguousConditionEdit = errtext.Sentence("Escolha o campo ou a perícia do teste, não os dois.")
	// ErrDamageHasNoAdvantage means a condition edit set a bias on the damage. Damage rolls a
	// single set of dice (rollActionDice: "damage has no advantage"), so there is no second set
	// for a bias to choose — accepting it would store something no reading can ever use. The
	// flat modifier is accepted.
	ErrDamageHasNoAdvantage = errtext.Sentence("O dano não aceita vantagem nem desvantagem.")
	// ErrCharacterAlreadyInSession means AddNPC was asked to add a character that already has
	// combat state in this session — a second AddNPC for the same sheetUUID, or one that
	// collides with an existing player character. It is an error rather than a silent
	// no-op/overwrite: overwriting would reset an NPC that has already taken damage or acted.
	ErrCharacterAlreadyInSession = errtext.Sentence("Esse personagem já está na partida.")
	// ErrInvalidNPC means AddNPC was called with a Nil sheetUUID, a nil sheet or a Nil
	// masterUUID. None of the three has a sane fallback, and writing partial state from one
	// of them would corrupt charSheets/statuses/charToPlayer for a character nobody can
	// address afterwards.
	ErrInvalidNPC = errtext.Sentence("Faltam dados na ficha desse NPC para colocá-lo na partida.")
	// ErrNotAnEscape means an escapeLanding named something that does not displace — the
	// turn's own action, or a reaction whose kind is not one of the three escapes. Only an
	// escape has a piece whose landing the master can decide.
	ErrNotAnEscape = errtext.Sentence("O pouso só vale para uma fuga no turno aberto.")
	// ErrLandingOutOfGrid means the master put a failed escape's piece on a cell the board
	// does not have.
	ErrLandingOutOfGrid = errtext.Sentence("Esse pouso fica fora do mapa.")
	// ErrNoDamageToMeasure means a damageSkill edit named an action that carries no attack —
	// there is no damage for a skill to measure.
	ErrNoDamageToMeasure = errtext.Sentence("Essa ação não tem ataque, então não há dano para medir.")
	// ErrQueuedEditUnsupported means an edit_action on a QUEUED action tried to change its
	// targets or its skill list. See ApplyMasterAction for what lifting it takes.
	ErrQueuedEditUnsupported = errtext.Sentence("Uma ação na fila só aceita mudar as condições e a perícia de dano.")
	// ErrStackedOnPending: o pull_action nomeou um movimento empilhado num movimento do mesmo
	// personagem que ainda está na fila (spec 2026-11-06 D4). Um empilhado nunca abre antes do
	// anterior; puxe a cabeça primeiro. Nada muda: nenhum turno fecha, nenhum preço congela.
	ErrStackedOnPending = errtext.Sentence("Abra antes o movimento de que este depende.")
	// ErrMatchInBattle: encerrar a partida com a cena atual em combate. Troque de cena antes.
	ErrMatchInBattle = errtext.Sentence("Encerre o combate antes de encerrar a partida.")
	// ErrMatchEnded: a partida já foi encerrada.
	ErrMatchEnded = errtext.Sentence("A partida já terminou.")
	// ErrBriefFinalTooLong: o resumo final passa de 255 caracteres.
	ErrBriefFinalTooLong = errtext.Sentence("O resumo final deve ter no máximo 255 caracteres.")
	// ErrBlockedByStatus: um status do personagem proíbe a action ou a reação (StatusRefusal). A
	// mensagem diz o que está proibido e por quais tipos ("<personagem> não pode agir: Atordoado", pelo nome da mesa).
	ErrBlockedByStatus = errors.New("blocked by status")
)
