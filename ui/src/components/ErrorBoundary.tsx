import React from "react";
import { Component } from "react";

export class ErrorBoundary extends Component<{ children: React.ReactNode }> {
  state = { error: false, errorMessage: "" };

  static getDerivedStateFromError(error: any) {
    return { error: true, errorMessage: error.toString() };
  }

  componentDidCatch(error: any) {
    console.error(error);
  }

  render() {
    if (this.state.error) {
      return (
        <div className="alert alert-danger" role="alert">
          <h4 className="alert-heading">Failed</h4>
          {this.state.errorMessage}
        </div>
      );
    }
    return this.props.children;
  }
}
