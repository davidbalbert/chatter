package config

type graph struct {
	nodes map[ServiceID][]ServiceID
}

func newGraph() *graph {
	return &graph{nodes: make(map[ServiceID][]ServiceID)}
}

func (g *graph) addNode(id ServiceID, deps ...ServiceID) {
	g.nodes[id] = deps
}

func (g *graph) topologicalSort() []ServiceID {
	visited := make(map[ServiceID]bool)
	stack := []ServiceID{}

	var visit func(ServiceID)

	visit = func(service ServiceID) {
		if _, ok := visited[service]; !ok {
			visited[service] = true

			for _, dep := range g.nodes[service] {
				visit(dep)
			}

			stack = append(stack, service)
		}
	}

	for service := range g.nodes {
		visit(service)
	}

	return stack
}
