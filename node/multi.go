package node

import "fmt"

// execCommandAbort Aborts a transaction, with specific error message.
// The transaction is always aborted with -EXECABORT so that the client knows
func (c *Client) execCommandAbort(s string) {
	c.addReplyErrorFormat(fmt.Sprintf("-EXECABORT Transaction discarded because of: %s", s))
}
