package parser

// Parses an INSERT statement.
//
// The generated AST is as follows:
//
//	|-> "INSERT" (InsertToken)
//	    |-> "INTO" (IntoToken)
//	        |-> table name
//	            |-> column name
//	            |-> (...)
//	    |-> "VALUES" (ValuesToken)
//	        |-> "(" (BracketOpeningToken)
//	            |-> value
//	            |-> (...)
//	        |-> (...)
//	    |-> "ON" (OnToken) (optional)
//	        |-> "CONFLICT" (ConflictToken)
//	            |-> (optional) column name
//	            |-> "DO" (DoToken)
//	                |-> "UPDATE" (UpdateToken)
//	                    |-> "SET" (SetToken)
//	                        |-> column name
//	                        |-> (...)
//	    |-> "RETURNING" (ReturningToken) (optional)
//	        |-> column name
func (p *parser) parseInsert() (*Instruction, error) {
	i := &Instruction{}

	// Set INSERT decl
	insertDecl, err := p.consumeToken(InsertToken)
	if err != nil {
		return nil, err
	}
	i.Decls = append(i.Decls, insertDecl)

	// should be INTO
	intoDecl, err := p.consumeToken(IntoToken)
	if err != nil {
		return nil, err
	}
	insertDecl.Add(intoDecl)

	// should be table Name
	tableDecl, err := p.parseTableName()
	if err != nil {
		return nil, err
	}
	intoDecl.Add(tableDecl)

	_, err = p.consumeToken(BracketOpeningToken)
	if err != nil {
		return nil, err
	}

	// concerned attribute
	for {
		decl, err := p.parseListElement()
		if err != nil {
			return nil, err
		}
		tableDecl.Add(decl)

		if p.is(BracketClosingToken) {
			if _, err = p.consumeToken(BracketClosingToken); err != nil {
				return nil, err
			}

			break
		}

		_, err = p.consumeToken(CommaToken)
		if err != nil {
			return nil, err
		}
	}

	// should be VALUES
	valuesDecl, err := p.consumeToken(ValuesToken)
	if err != nil {
		return nil, err
	}
	insertDecl.Add(valuesDecl)

	for {
		openingBracketDecl, err := p.consumeToken(BracketOpeningToken)
		if err != nil {
			return nil, err
		}
		valuesDecl.Add(openingBracketDecl)

		// should be a list of values for specified attributes
		for {
			decl, err := p.parseListElement()
			if err != nil {
				return nil, err
			}
			openingBracketDecl.Add(decl)

			if p.is(BracketClosingToken) {
				p.consumeToken(BracketClosingToken)
				break
			}

			_, err = p.consumeToken(CommaToken)
			if err != nil {
				return nil, err
			}
		}

		if p.is(CommaToken) {
			p.consumeToken(CommaToken)
			continue
		}

		break
	}

	// we may have `ON CONFLICT DO UPDATE` here
	if onDecl, err := p.consumeToken(OnToken); err == nil {
		insertDecl.Add(onDecl)

		// should be CONFLICT
		conflictDecl, err := p.consumeToken(ConflictToken)
		if err != nil {
			return nil, err
		}
		onDecl.Add(conflictDecl)

		// Optional: (column, ...)
		if p.is(BracketOpeningToken) {
			_, err := p.consumeToken(BracketOpeningToken)
			if err != nil {
				return nil, err
			}

			for {
				decl, err := p.parseAttribute()
				if err != nil {
					return nil, err
				}
				conflictDecl.Add(decl)

				if p.is(BracketClosingToken) {
					_, err := p.consumeToken(BracketClosingToken)
					if err != nil {
						return nil, err
					}
					break
				}

				_, err = p.consumeToken(CommaToken)
				if err != nil {
					return nil, err
				}
			}
		}

		// should be DO
		doDecl, err := p.consumeToken(DoToken)
		if err != nil {
			return nil, err
		}
		conflictDecl.Add(doDecl)

		if p.is(UpdateToken) {
			// should be UPDATE
			updateDecl, err := p.consumeToken(UpdateToken)
			if err != nil {
				return nil, err
			}
			doDecl.Add(updateDecl)

			// should be SET
			setDecl, err := p.consumeToken(SetToken)
			if err != nil {
				return nil, err
			}
			updateDecl.Add(setDecl)

			// should be a list of equality
			for {
				attributeDecl, err := p.parseAttribution()
				if err != nil {
					return nil, err
				}
				setDecl.Add(attributeDecl)

				if !p.is(CommaToken) {
					break
				}
				if _, err := p.consumeToken(CommaToken); err != nil {
					return nil, err
				}
			}
		} else if p.is(NothingToken) {
			nothingDecl, err := p.consumeToken(NothingToken)
			if err != nil {
				return nil, err
			}
			doDecl.Add(nothingDecl)
		} else {
			return nil, p.syntaxError()
		}
	}

	// we may have `returning "something"` here
	if retDecl, err := p.consumeToken(ReturningToken); err == nil {
		insertDecl.Add(retDecl)

		// returned attribute
		attrDecl, err := p.parseAttribute()
		if err != nil {
			return nil, err
		}
		retDecl.Add(attrDecl)
	}

	return i, nil
}

func (p *parser) parseListElement() (*Decl, error) {
	quoted := false

	// In case of INSERT, can be DEFAULT here
	if p.is(DefaultToken) {
		v, err := p.consumeToken(DefaultToken)
		if err != nil {
			return nil, err
		}
		return v, nil
	}

	if p.is(SimpleQuoteToken) || p.is(DoubleQuoteToken) {
		quoted = true
		p.next()
	}

	var valueDecl *Decl
	valueDecl, err := p.consumeToken(FloatToken, StringToken, NumberToken, NullToken, DateToken, NowToken, ArgToken, NamedArgToken, FalseToken)
	if err != nil {
		return nil, err
	}

	if quoted {
		if _, err := p.consumeToken(SimpleQuoteToken, DoubleQuoteToken); err != nil {
			return nil, err
		}
	}

	return valueDecl, nil
}
