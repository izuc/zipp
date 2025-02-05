import * as React from 'react';
import NodeStore from "app/stores/NodeStore";
import Card from "react-bootstrap/Card";
import {Link} from 'react-router-dom';
import {inject, observer} from "mobx-react";
import * as dateformat from 'dateformat';

interface Props {
    nodeStore?: NodeStore;
}

@inject("nodeStore")
@observer
export default class MeshTime extends React.Component<Props, any> {
    render() {
        return (
            <Card>
                <Card.Body>
                    <Card.Title>MeshTime
                        Synced: {this.props.nodeStore.status.meshTime.synced ? "Yes" : "No"}</Card.Title>
                    <small>
                        <div>
                            <hr/>
                            <div className={"row"}>
                                <div className={"col-12"}>Last Accepted Block: <Link
                                    to={`/explorer/block/${this.props.nodeStore.status.meshTime.acceptedBlockID}`}>
                                    {this.props.nodeStore.status.meshTime.acceptedBlockID}
                                </Link></div>
                            </div>
                            <div className={"row"}>
                                <div className={"col-12"}>Last Confirmed Block: <Link
                                    to={`/explorer/block/${this.props.nodeStore.status.meshTime.confirmedBlockID}`}>
                                    {this.props.nodeStore.status.meshTime.confirmedBlockID}
                                </Link></div>
                            </div>
                            <hr/>
                            <div className={"row"}>
                                <div className={"col-3"}>
                                    Acceptance Time:
                                </div>
                                <div className={"col-3"}>
                                    {dateformat(new Date(this.props.nodeStore.status.meshTime.ATT / 1000000), "dd.mm.yyyy HH:MM:ss")}
                                </div>
                                <div className={"col-3"}>
                                    Confirmation Time:
                                </div>
                                <div className={"col-3"}>
                                    {dateformat(new Date(this.props.nodeStore.status.meshTime.CTT / 1000000), "dd.mm.yyyy HH:MM:ss")}
                                </div>
                            </div>
                            <div className={"row"}>
                                <div className={"col-3"}>
                                    Relative Acceptance Time:
                                </div>
                                <div className={"col-3"}>
                                    {dateformat(new Date(this.props.nodeStore.status.meshTime.RATT / 1000000), "dd.mm.yyyy HH:MM:ss")}
                                </div>
                                <div className={"col-3"}>
                                    Relative Confirmation Time:
                                </div>
                                <div className={"col-3"}>
                                    {dateformat(new Date(this.props.nodeStore.status.meshTime.RCTT / 1000000), "dd.mm.yyyy HH:MM:ss")}
                                </div>
                            </div>
                        </div>
                    </small>
                </Card.Body>
            </Card>
        )
            ;
    }
}
