import { useState, useEffect } from 'react';
import { listProcesses, killProcess } from '../services/api';
import './ProcessPanel.css';

const ProcessPanel = () => {
  const [processes, setProcesses] = useState([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState(null);

  const loadProcesses = async () => {
    try {
      setIsLoading(true);
      setError(null);
      const procs = await listProcesses();
      setProcesses(procs);
    } catch (err) {
      setError(err.message);
      console.error('Failed to load processes:', err);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadProcesses();
    // Refresh every 2 seconds
    const interval = setInterval(loadProcesses, 2000);
    return () => clearInterval(interval);
  }, []);

  const handleKillProcess = async (pid) => {
    if (!confirm(`Are you sure you want to kill process ${pid}?`)) {
      return;
    }

    try {
      await killProcess(pid);
      await loadProcesses(); // Refresh list
    } catch (err) {
      alert(`Failed to kill process: ${err.message}`);
      console.error('Failed to kill process:', err);
    }
  };

  const formatDuration = (startTime) => {
    const start = new Date(startTime);
    const now = new Date();
    const diffMs = now - start;
    const diffSec = Math.floor(diffMs / 1000);
    const diffMin = Math.floor(diffSec / 60);
    const diffHour = Math.floor(diffMin / 60);

    if (diffHour > 0) {
      return `${diffHour}h ${diffMin % 60}m`;
    } else if (diffMin > 0) {
      return `${diffMin}m ${diffSec % 60}s`;
    } else {
      return `${diffSec}s`;
    }
  };

  return (
    <div className="process-panel">
      <div className="process-panel-header">
        <h2>Background Processes</h2>
        <button onClick={loadProcesses} className="refresh-btn" disabled={isLoading}>
          {isLoading ? '...' : '↻'}
        </button>
      </div>

      {error && (
        <div className="process-panel-error">
          Error: {error}
        </div>
      )}

      <div className="process-list">
        {processes.length === 0 ? (
          <div className="process-empty">
            No background processes running
          </div>
        ) : (
          processes.map((proc) => (
            <div key={proc.pid} className="process-item">
              <div className="process-info">
                <div className="process-pid">PID: {proc.pid}</div>
                <div className="process-command" title={proc.command}>
                  {proc.command.length > 50 
                    ? `${proc.command.substring(0, 50)}...` 
                    : proc.command}
                </div>
                <div className="process-meta">
                  <span className="process-duration">
                    Running: {formatDuration(proc.start_time)}
                  </span>
                  {proc.conversation_id && (
                    <span className="process-conv-id">
                      Conv: {proc.conversation_id.substring(0, 8)}...
                    </span>
                  )}
                </div>
              </div>
              <button
                className="kill-btn"
                onClick={() => handleKillProcess(proc.pid)}
                title="Kill process"
              >
                ×
              </button>
            </div>
          ))
        )}
      </div>
    </div>
  );
};

export default ProcessPanel;

