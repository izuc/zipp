import * as ReactDOM from 'react-dom';
import { Provider } from 'mobx-react';
import MeshStore from 'stores/MeshStore';
import UTXOStore from 'stores/UTXOStore';
import ConflictStore from 'stores/ConflictStore';
import GlobalStore from 'stores/GlobalStore';
import { Root } from 'components/Root';
import React from 'react';

const meshStore = new MeshStore();
const utxoStore = new UTXOStore();
const conflictStore = new ConflictStore();
const globalStore = new GlobalStore(meshStore, utxoStore, conflictStore);

const stores = {
    meshStore: meshStore,
    utxoStore: utxoStore,
    conflictStore: conflictStore,
    globalStore: globalStore
};

ReactDOM.render(
    <Provider {...stores}>
        <Root />
    </Provider>,
    document.getElementById('root')
);
