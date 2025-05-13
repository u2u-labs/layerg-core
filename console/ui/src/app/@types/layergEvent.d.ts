export interface IListNode {
  errorMessage: string;
  message: string;
  nodes: INode[];
  success: boolean;
}

export interface INode {
  domain: string;
  node: string;
  chainId: string;
}

export interface INodeForChain {
  errorMessage: string;
  nodeAddress: string;
  nodeDomain: string;
}
