import {environment} from "./environments/environment";

type AbiEventParam = {
  type: string;
};

type AbiEvent = {
  type: string;
  name: string;
  inputs?: AbiEventParam[];
};

export function parseEventSignaturesOnly(abi: any[]): string[] {
  return abi
    .filter((item: AbiEvent) => item.type === 'event')
    .map((event: AbiEvent) => {
      const types = (event.inputs || []).map((input) => input.type).join(', ');
      return `${event.name}(${types})`;
    });
}

export const CHAIN_IDS = [
  { name: 'U2U Network (Testnet)', id: 2484 },
  { name: 'U2U Network (Mainnet)', id: 39 },
];
