import * as React from 'react';
import Container from 'react-bootstrap/Container';
import Row from 'react-bootstrap/Row';
import MeshDAG from 'components/MeshDAG';
import UTXODAG from 'components/UTXODAG';
import ConflictDAG from 'components/ConflictDAG';
import GlobalSettings from 'components/GlobalSettings';
import {connectWebSocket} from 'utils/WS';
import {Navbar} from 'react-bootstrap';
import logo from './../images/logo_dark.png';

export class Root extends React.Component {
    connect = () => {
        connectWebSocket(
            '/ws',
            () => {
                console.log('connection opened');
            },
            this.reconnect,
            () => {
                console.log('connection error');
            }
        );
    };

    reconnect = () => {
        setTimeout(() => {
            this.connect();
        }, 1000);
    };

    componentDidMount(): void {
        this.connect();
    }

    render() {
        return (
            <>
                <Navbar className={'nav'}>
                    <img
                        src={logo}
                        alt={'DAGs Visualizer'}
                        style={{ height: '50px' }}
                    />
                </Navbar>
                <Container>
                    <Row>
                        <GlobalSettings />
                    </Row>
                    <Row>
                        <MeshDAG />
                    </Row>
                    <Row>
                        <UTXODAG />
                    </Row>
                    <Row>
                        <ConflictDAG />
                    </Row>
                </Container>
            </>
        );
    }
}
