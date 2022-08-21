package app

type Singletons map[string]interface{}

func (s Singletons) Monitor() *monitorServers {
	return &monitorServers{Singletons: s}
}

func (s Singletons) MockStart() MockRuntime {
	r := MockRuntime{}
	s["monitor"] = s.Monitor()
	for _, s := range s {
		c, ok := s.(configurable)
		if !ok {
			continue
		}
		err := c.Configure(nil)
		if err != nil {
			panic(err)
		}
	}
	for k, v := range s {
		service, ok := v.(Service)
		if !ok {
			continue
		}
		ctx := MockCtx()
		ctx.name = k
		service.Start(ctx)
		r[k] = ctx
	}
	return r
}
