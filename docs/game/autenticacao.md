# Autenticação e Usuários

## Registro

Para utilizar a plataforma, o jogador deve criar uma conta com:

| Campo | Regras |
|-------|--------|
| Nick | Obrigatório, 3–20 caracteres, único |
| Email | Obrigatório, 12–64 caracteres, único |
| Senha | Obrigatória, 8–64 caracteres |
| Confirmação de Senha | Deve ser idêntica à senha |

### Validações
- Nick e email são verificados por unicidade no banco
- Senhas são armazenadas com hash (bcrypt)

## Login

O login é feito via email + senha:

1. Validações de formato (email e senha com tamanhos válidos)
2. Busca do usuário pelo email
3. Verificação da senha via bcrypt
4. Geração de token JWT
5. Armazenamento da sessão (em memória + persistência)

### Sessão
- Token JWT é gerado com o UUID do usuário
- Sessão é armazenada em `sync.Map` (memória) para acesso rápido
- Sessão também é persistida no banco para recuperação após restart
- Se a persistência falhar, o login ainda é bem-sucedido (fire-and-forget)

### Erros
- Email não encontrado → "access denied" (não revela se o email existe)
- Senha incorreta → "access denied" (mesma mensagem, por segurança)

## Papéis na Plataforma

| Papel | Descrição |
|-------|-----------|
| Mestre | Cria cenários, campanhas e partidas. Aceita/rejeita fichas. |
| Jogador | Cria fichas, submete para campanhas, inscreve em partidas. |

> **Nota:** Atualmente não há distinção explícita de papéis no sistema. O mesmo usuário pode ser mestre em uma campanha e jogador em outra.
