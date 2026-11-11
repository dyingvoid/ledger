import grpc from 'k6/net/grpc';

const client = new grpc.Client();

export const options = {
  scenarios: {
    probe: { executor: 'shared-iterations', vus: 1, iterations: 1 },
  },
};

export default function () {
  client.connect('127.0.0.1:55051', { plaintext: true, reflect: true });
  const response = client.invoke(
    'ledger.funds.v1.FundsService/Deposit',
    {
      id: '0x' + Date.now().toString(16).padStart(12, '0') + '000000000000000001',
      account_id: '900000',
      amount_minor: 100,
      currency: 'USD',
    }
  );
  const s = response.status;
  console.log('STATUS=', s, 'TYPE=', typeof s, 'STRING=', String(s), 'NUMBER=', Number(s));
  console.log('KEYS=', JSON.stringify(Object.keys(s || {})));
  console.log('STATUS_STR=', typeof s.string === 'function' ? s.string() : 'n/a');
  console.log('EQ5=', s == 5, 'STRICT5=', s === 5, 'EQ_OK_CONST=', s === grpc.StatusOK);
  console.log('OK_CONST=', grpc.StatusOK, typeof grpc.StatusOK);
  console.log('ERR_CODE=', response.error && response.error.code, 'ERR_CODE_TYPE=', typeof (response.error && response.error.code));
  console.log('ERROR=', JSON.stringify(response.error));
  console.log('MESSAGE=', JSON.stringify(response.message));
}
