package charactersheet

import "context"

type IRepository interface {
	Test(ctx context.Context, data string) error
}
