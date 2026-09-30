describe('Reply-only workspace', () => {
  beforeEach(() => {
    cy.login()
    cy.visit('/addresses')
  })

  it('does not offer a new outbound conversation control', () => {
    cy.contains('New conversation').should('not.exist')
  })
})
