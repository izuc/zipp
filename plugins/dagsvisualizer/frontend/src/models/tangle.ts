export class meshVertex {
    ID: string;
    strongParentIDs: Array<string>;
    weakParentIDs: Array<string>;
    shallowLikeParentIDs: Array<string>;
    conflictIDs: Array<string>;
    isMarker: boolean;
    isTx: boolean;
    txID: string;
    isTip: boolean;
    isConfirmed: boolean;
    isTxConfirmed: boolean;
    confirmationState: string;
    confirmationStateTime: number;
}

export class meshBooked {
    ID: string;
    isMarker: boolean;
    conflictIDs: Array<string>;
}

export class meshConfirmed {
    ID: string;
    confirmationState: string;
    confirmationStateTime: number;
}

export class meshTxConfirmationStateChanged {
    ID: string;
    isConfirmed: boolean;
}

export enum parentRefType {
    StrongRef,
    WeakRef,
    ShallowLikeRef,
    ShallowDislikeRef
}
