import { useMemo, useState } from 'react';

export function App() {
  const [count, setCount] = useState(0);
  const loadedAt = useMemo(() => new Date().toISOString(), []);

  return (
    <div className="page">
      <h1>k8s-lab UI (Vite + React)</h1>
      <p className="muted">Served via Nginx, routed through Envoy Gateway at /vite.</p>

      <div className="card">
        <div className="row">
          <span className="label">Loaded at</span>
          <code>{loadedAt}</code>
        </div>
        <div className="row">
          <span className="label">Clicks</span>
          <code>{count}</code>
        </div>
        <button type="button" onClick={() => setCount((c) => c + 1)}>
          Click me
        </button>
      </div>
    </div>
  );
}
