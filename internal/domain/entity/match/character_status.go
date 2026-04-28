package match

import "github.com/422UR4H/HxH_RPG_System/internal/domain/entity/match/action"

// tentativa de centralizar todos os status dos personagens
// pensei em trocar de CharacterStatus para Field em Match,
// mas lembrei que existem/existirão outros status:
// - posições: abaixado, deitado, caído
//   - talvez "caído" possa ser combinação de deitado com atordoado
//
// - outros: paralizado, atordoado, inconsciente, furtivo?
// - stamina: cansado, exausto,
// - sentidos: cego, surdo, "olfato comprometido", - talvez mudo também
// - mental: confuso, abalado, cansado, exausto, insano
type CharacterStatus struct {
	Position  [3]int // x, y, z
	Velocity  action.Velocity
	MoveBar   int
	ActionBar int
	// Facing enum.Facing // direção que o personagem está olhando - futuro
	// Stance enum.Stance // posição, postura
	// Status []enum.MatchStatus
}

// O CharacterStatus não será persistido
// ele precisa ser construído dinamicamente à partir das actions
// e precisa ser recriado à partir delas
// - considerar statusEngine, ou algo assim
//
// Racional - barra de ação e de movimento:
// MoveBar e ActionBar são os pontos atuais (ou bônus) de cada barra.
// cada ação (de movimento ou genérica) tem um custo em pontos.
// esse custo é subtraído da barra quando esta ação é realizada.
//
//
// fazendo com que ela seja realizada,
// quando o mestre "abre" o turno, a ação com maior valor na barra é retirada da fila
//
//
//
//
// Racional - movimento:
// a forma correta de se pensar no movimento seria:
// turno 0, todo mundo está parado
// * (isso não é verdade absoluta, mas vamos começar assim)
// - todo mundo indica sua inteção de ação e de movimento
// - e começa a se mover, de fato, no momento que sua ação abre
// - mas o movimento não deve ser concluído nesse instante,
// - a "peça" só deveria "terminar de se mover" na próxima move action
// - então qualquer ação "amarrada" no move só deveria ocorrer nesse momento
// 	- claro que seria possível se mover e agir ao mesmo tempo,
// 	- mas arremeditas\investidas só deveriam acontecer no "próximo turno"
//
// - dessa forma, na abertura da move action, apareceria o vetor velocidade
// - mas a peça só deveria se mover, de fato, da metade pro fim do movimento
//
//--> não sei se esse racional acima cobre todos os casos
//	-> mas arremetidas\investidas só devem acontecer logo após o movimento mesmo
//
//
//
// outra forma de se pensar no movimento que também seria correto:
// - no tópico, "a peça só deveria se mover da metade pro fim do movimento"
// - percebemos que pode existir uma "aproximação" desse movimento
// - podemos utilizar essa percepção de aproximação para não criar esse "lag"
// - então, podemos perceber da seguinte forma:
// - aproximamos o quickness (início do movimento) de accelerate\brake (vel)
// - usamos a média entre eles para decidir quando o personagem age, de fato
// * consideramos o início do movimento, mas também o "fim" de "cada parte dele"
// 	- porque cada "slot" seria uma "parte do movimento"
// -> com a velocidade, podemos saber "quando" o personagem se moveria novamente
//	- porque ela que determina a próxima ação de movimento do personagem
// 	- mesmo que ele decida fazer um dash (primeiro) seguido de um shift
//	- a velocidade só começaria a se alterar (diminuir) na próxima move action
// 	- logo, temos os parâmetros necessários para realizar essa aproximação
// -> com ela, poderíamos realizar o movimento no instante da move action
// - removendo o lag e a inconsistência por meio do erro assumido na aproximação
//	- nesse caso, o deslocamento da peça ocorre no "meio tempo" desse movimento
//	- ou seja, no instante que a peça cruzaria a linha entre os slots
//	- assim também poderíamos considerar que o personagem não precisaria estar
//	- exatamente no slot adjascente ao alvo para desferir o golpe
//	* o problema começa, caso o target (alvo) esteja se afastando do actor
//		x nesse caso, o sistema precisa verificar isso e notificar\permitir mudança
//		x ou seja, o actor deveria poder mudar a action (alvo\ataque), se quisesse
//		x isso pode acontecer através de uma verificação se os alvos estão melee
//			- porque, obviamente, essa restrição se limita apenas para ataques melee
//  	-> ideia melhor, se o alvo estiver se movendo, todos veem através da seta
//			- então o actor decidiu fazer isso de propósito.
//			- nesse caso, considerando que em algum instante no board, eles "colam"
//			- assim, o alvo pode tentar um scape, se quiser
//			- dando um dash para frente, por exemplo. fato é que ele tem vantagem
//			- ele pode até escolher uma opção que não consolide essa vantagem,
//			- mas ele sabe o que pode fazer e decidir livremente
//			- de qualquer forma o mestre decidirá no final, também.
//
//
//
// - dessa forma, arremetidas\investidas só ocorrem imediatamente após a move,
// - mas não apenas no final do movimento, e sim, na "metade aproximada" dele,
// - e as actions "desamarradas" da move podem acontecer antes
//	- no caso da action speed ser mais rápida que a move speed
//
// com essa aproximação, poderemos explorar ações e casos específicos
// que só considerariam quickness para o movimento
//
// nota: uma vez que se tenha começado a mover, o que conta é a velocidade
// - considerar isso na aproximação
//
//
//	A Iniciativa soma apenas na primeira action do turno para cada personagem
//
//
//
//	-> o movimento depende de quickness, então acaba sendo mais lento
//		-> ordem: reflexo (iniciativa), quickness, accelerate
//
//
//
//	a velocidade de uma ação específica (ActionSpeed) é calculada da ActionSpeed
//	que define o valor da barra nesse turno
//		- se o personagem conseguir realizar mais um ataque,
//			o ActionSpeed do próximo turno é rolado!
//			e a média entre os turnos é o ActionSpeed do ataque
//			- se for mais de uma ação excedente também usa a média
//				- decisão tomada porque a curva linear não necessáriamente aproxima bem
//					então não vale o esforço e a complexidade maior
//
//
//
//
//
// pensar na mecânica de clash:
// - caso onde personagens disputam o slot através de testes de perícia
// - cada player escolhe\pode escolher sua perícia e narrar sua ação,
// - que é validada pelo mestre (no fluxo) se as descrições e escolhas condizem
// 	- essa mecânica pode permitir "roubar" turno de ação.
// * esse botão também deve aparecer ao clicar no alvo
//	- ele pode funcionar como investida também,
//	- de forma que o personagem precisa saltar, se estiver 1+ slots de distância
//	- isso, por si só, talvez já devesse ser um "charge"
//		- tanto por conta do salto, quanto da gravidade -> desenvolver melhor
// 	- se falhar de alguma forma, o mestre decide onde ele cai
//	- mas o mais comum, imagino que seria o alvo realizar scape
//	* ex.: ataque do Silva no Kuroro, logo antes de ser cortado pela faca Benz
//
// outra mecânica seria a de se movem para o mesmo slot sem disputá-lo
// - esta seria realizada exclusivamente por meio de Flexibilidade:
//	- Acrobatics, Sneak ou até mesmo Evasion
// - escolhido pelo actor player e validado pelo mestre, também
//
//
// -> pensar na mecânica de footwork
//
// outra mecânica seria a de derrubar o alvo
// - o jogo de pés (Footwork) é um teste de Quickness,
//   - necessário para se mover para o slot ocupado pelo alvo
//   - a partir daí vai depender dos detalhes da action e o teste do alvo também
// - para configurar essa action, o actor deve escolher um movimento
// --  - que pode ser um Dash, Shift ou Step (ou até mesmo um Leap, dependendo)
// --  - e uma ação de ataque corpo a corpo (Melee Attack)
//	 - em seguida, ataque corporal
// - se bem executado, o
