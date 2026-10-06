import React from 'react'

type Props = { children: React.ReactNode }
type State = { error: Error | null }

export default class ErrorBoundary extends React.Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error('UI crash:', error, info)
  }

  render() {
    if (this.state.error) {
      return (
        <div className="auth-page">
          <div className="auth-card">
            <h1>Something went wrong</h1>
            <p className="sub">The page hit an error and could not render. Try a hard refresh.</p>
            <div className="error" style={{ wordBreak: 'break-word' }}>
              {this.state.error.message}
            </div>
            <div className="row-actions" style={{ marginTop: '1rem' }}>
              <button className="btn" type="button" onClick={() => window.location.reload()}>
                Reload page
              </button>
              <button
                className="btn secondary"
                type="button"
                onClick={() => this.setState({ error: null })}
              >
                Try again
              </button>
            </div>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}
