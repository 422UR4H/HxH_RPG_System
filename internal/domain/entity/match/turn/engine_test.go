package turn

import (
	"testing"

	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/enum"
	"github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"
	"github.com/google/uuid"
)

// Helper function para criar actions de teste
func createTestAction(speed int) *action.Action {
	return action.NewAction(
		uuid.New(),
		[]uuid.UUID{uuid.New()},
		uuid.Nil,
		[]action.Skill{},
		action.RollCheck{Result: speed},
		nil,
		nil,
		nil,
		nil,
		nil,
	)
}

func TestNewEngine(t *testing.T) {
	t.Run("cria engine com valores padrão", func(t *testing.T) {
		closeTurnTriggered := false
		engine := NewEngine(nil, nil, &closeTurnTriggered)

		if engine.actionQueue.Len() != 0 {
			t.Errorf("actionQueue deveria estar vazia, mas tem %d items", engine.actionQueue.Len())
		}

		if len(engine.turns) != 1 {
			t.Errorf("Deveria ter 1 turn padrão, mas tem %d", len(engine.turns))
		}
	})

	t.Run("cria engine com actions iniciais", func(t *testing.T) {
		actions := []*action.Action{
			createTestAction(10),
			createTestAction(5),
		}

		closeTurnTriggered := false
		engine := NewEngine(&actions, nil, &closeTurnTriggered)

		if engine.actionQueue.Len() != 2 {
			t.Errorf("actionQueue deveria ter 2 items, mas tem %d", engine.actionQueue.Len())
		}

		// Verifica se está ordenada processando a primeira (maior velocidade primeiro)
		first := engine.NextAction()
		if first.Speed.Result != 10 {
			t.Errorf("Primeira action deveria ter velocidade 10, mas tem %d", first.Speed.Result)
		}
	})
}

func TestEngineAddAction(t *testing.T) {
	closeTurnTriggered := false
	engine := NewEngine(nil, nil, &closeTurnTriggered)

	t.Run("adiciona actions e mantém ordem de prioridade", func(t *testing.T) {
		// Adiciona actions em ordem aleatória
		actions := []*action.Action{
			createTestAction(5),
			createTestAction(15),
			createTestAction(10),
			createTestAction(20),
			createTestAction(8),
		}

		for _, act := range actions {
			engine.Add(act)
		}

		// Verifica se estão ordenadas corretamente (decrescente)
		expectedOrder := []int{20, 15, 10, 8, 5}

		for i, expected := range expectedOrder {
			action := engine.NextAction()
			if action == nil {
				t.Fatalf("Action %d não deveria ser nil", i)
			}
			if action.Speed.Result != expected {
				t.Errorf("Action %d deveria ter velocidade %d, mas tem %d", i, expected, action.Speed.Result)
			}
		}

		// Verifica se a fila está vazia agora
		if !engine.actionQueue.IsEmpty() {
			t.Error("actionQueue deveria estar vazia após remover todas as actions")
		}
	})
}

func TestEngineGetNextAction(t *testing.T) {
	closeTurnTriggered := false
	engine := NewEngine(nil, nil, &closeTurnTriggered)

	t.Run("retorna nil quando fila está vazia", func(t *testing.T) {
		action := engine.NextAction()
		if action != nil {
			t.Error("GetNextAction deveria retornar nil quando fila está vazia")
		}
	})

	t.Run("remove e retorna action com maior velocidade", func(t *testing.T) {
		engine.Add(createTestAction(10))
		engine.Add(createTestAction(5))

		// Primeira chamada deve retornar a de velocidade 10
		action := engine.NextAction()
		if action.Speed.Result != 10 {
			t.Errorf("Deveria retornar action com velocidade 10, mas retornou %d", action.Speed.Result)
		}

		// Segunda chamada deve retornar a de velocidade 5
		action = engine.NextAction()
		if action.Speed.Result != 5 {
			t.Errorf("Deveria retornar action com velocidade 5, mas retornou %d", action.Speed.Result)
		}

		// Terceira chamada deve retornar nil
		action = engine.NextAction()
		if action != nil {
			t.Error("Terceira chamada deveria retornar nil")
		}
	})
}

func TestEngineGetCurrentAction(t *testing.T) {
	closeTurnTriggered := false
	engine := NewEngine(nil, nil, &closeTurnTriggered)

	t.Run("retorna nil quando nenhuma action foi processada", func(t *testing.T) {
		action := engine.GetCurrentAction()
		if action != nil {
			t.Error("GetCurrentAction deveria retornar nil quando nenhuma action foi processada")
		}
	})

	t.Run("retorna a action atualmente em execução", func(t *testing.T) {
		engine.Add(createTestAction(10))
		engine.Add(createTestAction(15))

		// Processa a primeira action
		processed := engine.NextAction()
		if processed.Speed.Result != 15 {
			t.Errorf("NextAction deveria retornar velocidade 15, mas retornou %d", processed.Speed.Result)
		}

		// GetCurrentAction deve retornar a mesma action em execução
		current := engine.GetCurrentAction()
		if current.GetID() != processed.GetID() {
			t.Error("GetCurrentAction deveria retornar a action em execução")
		}
		if current.Speed.Result != 15 {
			t.Errorf("GetCurrentAction deveria retornar velocidade 15, mas retornou %d", current.Speed.Result)
		}

		// A fila deve ter 1 item restante
		if engine.actionQueue.Len() != 1 {
			t.Errorf("actionQueue deveria ter 1 item após processar uma, mas tem %d", engine.actionQueue.Len())
		}
	})
}

func TestEngineChangeMode(t *testing.T) {
	closeTurnTriggered := false
	engine := NewEngine(nil, nil, &closeTurnTriggered)

	t.Run("alterna entre modos Free e Race", func(t *testing.T) {
		// Engine começa com modo Free
		if engine.mode != enum.Free {
			t.Errorf("Engine deveria começar com modo Free, mas tem %v", engine.mode)
		}

		// Primeira chamada deve definir como Race
		engine.ChangeMode(nil)
		if engine.mode != enum.Race {
			t.Errorf("Modo deveria ser Race, mas é %v", engine.mode)
		}

		// Segunda chamada deve voltar para Free
		engine.ChangeMode(nil)
		if engine.mode != enum.Free {
			t.Errorf("Modo deveria ser Free, mas é %v", engine.mode)
		}

		// Terceira chamada deve voltar para Race
		engine.ChangeMode(nil)
		if engine.mode != enum.Race {
			t.Errorf("Modo deveria ser Race novamente, mas é %v", engine.mode)
		}
	})
}

func TestEngineCompleteWorkflow(t *testing.T) {
	closeTurnTriggered := false
	engine := NewEngine(nil, nil, &closeTurnTriggered)

	t.Run("fluxo completo de uso", func(t *testing.T) {
		// Adiciona múltiplas actions
		actions := []*action.Action{
			createTestAction(12),
			createTestAction(8),
			createTestAction(15),
			createTestAction(10),
		}

		for _, act := range actions {
			engine.Add(act)
		}

		// Verifica que a fila tem todas as actions
		if engine.actionQueue.Len() != 4 {
			t.Errorf("Fila deveria ter 4 actions, mas tem %d", engine.actionQueue.Len())
		}

		// Processa actions em ordem de prioridade
		expectedOrder := []int{15, 12, 10, 8}
		for i, expected := range expectedOrder {
			action := engine.NextAction()
			if action.Speed.Result != expected {
				t.Errorf("Action %d deveria ter velocidade %d, mas tem %d", i, expected, action.Speed.Result)
			}
		}

		// Verifica que a fila está vazia
		if !engine.actionQueue.IsEmpty() {
			t.Error("Fila deveria estar vazia após processar todas as actions")
		}

		// Adiciona nova action após limpar a fila
		engine.Add(createTestAction(25))
		newAction := engine.NextAction()
		if newAction.Speed.Result != 25 {
			t.Errorf("Deveria poder adicionar action com velocidade 25 após limpar fila, mas tem %d", newAction.Speed.Result)
		}
	})
}
