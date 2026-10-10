package ofx

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sampleOFX = `OFXHEADER:100
DATA:OFXSGML
VERSION:102
SECURITY:NONE
ENCODING:USASCII
CHARSET:1252
COMPRESSION:NONE
OLDFILEUID:NONE
NEWFILEUID:NONE

<OFX>
<SIGNONMSGSRSV1>
<SONRS>
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<DTSERVER>20261010120000[0:GMT]
<LANGUAGE>ENG
</SONRS>
</SIGNONMSGSRSV1>
<BANKMSGSRSV1>
<STMTTRNRS>
<TRNUID>1001
<STATUS>
<CODE>0
<SEVERITY>INFO
</STATUS>
<STMTRS>
<CURDEF>USD
<BANKACCTFROM>
<BANKID>123456789
<ACCTID>987654321
<ACCTTYPE>CHECKING
</BANKACCTFROM>
<BANKTRANLIST>
<DTSTART>20260901000000
<DTEND>20260930235959
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20260915120000
<TRNAMT>-85.50
<FITID>TXN001
<NAME>WHOLE FOODS MKT
<MEMO>GROCERIES
</STMTTRN>
<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20260920083000
<TRNAMT>3500.00
<FITID>TXN002
<NAME>ACME CORP PAYROLL
<MEMO>SALARY
</STMTTRN>
</BANKTRANLIST>
<LEDGERBAL>
<BALAMT>8414.50
<DTASOF>20260930235959
</LEDGERBAL>
</STMTRS>
</STMTTRNRS>
</BANKMSGSRSV1>
</OFX>`

func TestParseOFX(t *testing.T) {
	stmt, err := Parse(sampleOFX)
	require.NoError(t, err)
	assert.Equal(t, "USD", stmt.Currency)
	assert.Equal(t, "987654321", stmt.AccountID)
	require.Len(t, stmt.Transactions, 2)

	// First tx (debit)
	tx1 := stmt.Transactions[0]
	assert.Equal(t, "TXN001", tx1.FITID)
	assert.Equal(t, "WHOLE FOODS MKT", tx1.Payee)
	assert.Equal(t, "GROCERIES", tx1.Memo)
	assert.Equal(t, -85.50, tx1.Amount)
	assert.Equal(t, 85.50, tx1.AbsAmount())
	assert.True(t, tx1.IsDebit())
	assert.Equal(t, time.Date(2026, 9, 15, 0, 0, 0, 0, time.UTC), tx1.Date)

	// Second tx (credit)
	tx2 := stmt.Transactions[1]
	assert.Equal(t, "TXN002", tx2.FITID)
	assert.Equal(t, "ACME CORP PAYROLL", tx2.Payee)
	assert.Equal(t, 3500.00, tx2.Amount)
	assert.False(t, tx2.IsDebit())
}
