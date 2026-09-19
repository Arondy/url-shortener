package args

import "flag"

type Args struct {
	InMemory bool
}

func Parse() Args {
	inMemoryFlag := flag.Bool("in-mem", false, "use in-memory storage instead of PostgreSQL")
	flag.Parse()

	return Args{
		InMemory: *inMemoryFlag,
	}
}
