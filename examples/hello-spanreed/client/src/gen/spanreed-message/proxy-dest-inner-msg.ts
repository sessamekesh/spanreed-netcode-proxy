// automatically generated by the FlatBuffers compiler, do not modify

/* eslint-disable @typescript-eslint/no-unused-vars, @typescript-eslint/no-explicit-any, @typescript-eslint/no-non-null-assertion */

import { ProxyDestClientMessage } from '../spanreed-message/proxy-dest-client-message';
import { ProxyDestCloseConnection } from '../spanreed-message/proxy-dest-close-connection';
import { ProxyDestConnectionRequest } from '../spanreed-message/proxy-dest-connection-request';


export enum ProxyDestInnerMsg {
  NONE = 0,
  ProxyDestConnectionRequest = 1,
  ProxyDestClientMessage = 2,
  ProxyDestCloseConnection = 3
}

export function unionToProxyDestInnerMsg(
  type: ProxyDestInnerMsg,
  accessor: (obj:ProxyDestClientMessage|ProxyDestCloseConnection|ProxyDestConnectionRequest) => ProxyDestClientMessage|ProxyDestCloseConnection|ProxyDestConnectionRequest|null
): ProxyDestClientMessage|ProxyDestCloseConnection|ProxyDestConnectionRequest|null {
  switch(ProxyDestInnerMsg[type]) {
    case 'NONE': return null; 
    case 'ProxyDestConnectionRequest': return accessor(new ProxyDestConnectionRequest())! as ProxyDestConnectionRequest;
    case 'ProxyDestClientMessage': return accessor(new ProxyDestClientMessage())! as ProxyDestClientMessage;
    case 'ProxyDestCloseConnection': return accessor(new ProxyDestCloseConnection())! as ProxyDestCloseConnection;
    default: return null;
  }
}

export function unionListToProxyDestInnerMsg(
  type: ProxyDestInnerMsg, 
  accessor: (index: number, obj:ProxyDestClientMessage|ProxyDestCloseConnection|ProxyDestConnectionRequest) => ProxyDestClientMessage|ProxyDestCloseConnection|ProxyDestConnectionRequest|null, 
  index: number
): ProxyDestClientMessage|ProxyDestCloseConnection|ProxyDestConnectionRequest|null {
  switch(ProxyDestInnerMsg[type]) {
    case 'NONE': return null; 
    case 'ProxyDestConnectionRequest': return accessor(index, new ProxyDestConnectionRequest())! as ProxyDestConnectionRequest;
    case 'ProxyDestClientMessage': return accessor(index, new ProxyDestClientMessage())! as ProxyDestClientMessage;
    case 'ProxyDestCloseConnection': return accessor(index, new ProxyDestCloseConnection())! as ProxyDestCloseConnection;
    default: return null;
  }
}
