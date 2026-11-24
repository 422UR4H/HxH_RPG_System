package action

// RollAttempts guarda os dados que caíram para um teste: o conjunto da partida (Primary) e
// os dados extras de vantagem/desvantagem (Extra), um por nível.
//
// O conjunto cai uma vez, quando a ação ou reação chega (ou, num teste passivo, na primeira
// vez que um viés pede dados). Cada nível de vantagem ou desvantagem acrescenta um dado ao
// monte, e o resultado mantém os melhores (vantagem) ou os piores (desvantagem) do tamanho
// do conjunto. O mestre pode dar vantagem DEPOIS que os dados caíram, e nunca rola de novo o
// dado de um jogador: um viés que cresce tira só os extras que faltam, uma vez (TopUp), e um
// viés que diminui só deixa de ler os últimos extras — o que caiu fica.
//
// Mora neste pacote, e não no serviço de domínio, para que um RollCheck possa guardá-lo:
// service importa action, nunca o contrário. Sem tags json de propósito: linhas antigas
// gravaram "Primary"/"Secondary", e o "Secondary" delas é simplesmente ignorado na leitura.
type RollAttempts struct {
	Primary []int
	Extra   []int
}

// IsEmpty reports whether the dice have not fallen yet. It is what keeps the roll-once
// rule honest: whoever rolls checks this first and never overwrites a set that already
// landed.
func (a RollAttempts) IsEmpty() bool {
	return len(a.Primary) == 0 && len(a.Extra) == 0
}
