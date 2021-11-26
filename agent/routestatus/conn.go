package routestatus

type Connection struct {
	ConnectionID int      `json:"connection_id,omitempty"`
	GroupID      int      `json:"connection_group_id,omitempty"`
	RouteStatus  []*Entry `json:"statuses"`
}

func NewConnection(conID, grID int, entries ...*Entry) *Connection {
	c := &Connection{
		ConnectionID: conID,
		GroupID:      grID,
	}
	c.RouteStatus = append(c.RouteStatus, entries...)

	return c
}
