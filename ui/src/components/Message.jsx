import { useState } from 'react';
import './Message.css';

const Message = ({ message, toolMessages = [] }) => {
  const isUser = message.role === 'user';
  const isAssistant = message.role === 'assistant';
  const [expandedToolOutputs, setExpandedToolOutputs] = useState({});

  // Handle assistant messages that are calling tools
  let displayContent = message.content;
  let toolCallsDisplay = null;
  
  if (isAssistant && message.tool_calls && message.tool_calls.length > 0) {
    if (!displayContent) {
      displayContent = `calling ${message.tool_calls.length} ${message.tool_calls.length === 1 ? 'tool' : 'tools'}:`;
    }
    
    // Parse and display tool calls with their parameters
    toolCallsDisplay = (
      <div className="tool-calls-list">
        {message.tool_calls.map((toolCall, index) => {
          let parsedArgs = {};
          try {
            parsedArgs = JSON.parse(toolCall.arguments || '{}');
          } catch (e) {
            parsedArgs = { raw: toolCall.arguments };
          }
          
          // Find corresponding tool message output
          // Tool messages can have tool_call_id or TollCallID (Go struct field)
          const toolOutput = toolMessages.find(tm => {
            const tmCallId = tm.tool_call_id || tm.TollCallID;
            return tmCallId === toolCall.id;
          });
          const isExpanded = expandedToolOutputs[toolCall.id] || false;
          
          return (
            <div key={toolCall.id || index} className="tool-call-item">
              <div className="tool-call-header">
                <div className="tool-call-name">
                  <strong>{toolCall.name || 'tool'}</strong>
                </div>
              </div>
              <div className="tool-call-params">
                {Object.entries(parsedArgs).map(([key, value]) => (
                  <div key={key} className="tool-call-param">
                    <span className="param-key">{key}:</span>
                    <span className="param-value">{typeof value === 'string' ? value : JSON.stringify(value)}</span>
                  </div>
                ))}
              </div>
              {toolOutput && (
                <>
                  <button
                    className="tool-output-toggle"
                    onClick={() => setExpandedToolOutputs(prev => ({
                      ...prev,
                      [toolCall.id]: !prev[toolCall.id]
                    }))}
                    title={isExpanded ? 'Hide output' : 'Show output'}
                  >
                    {isExpanded ? '▼' : '▶'} Output
                  </button>
                  {isExpanded && (
                    <div className="tool-output">
                      <div className="tool-output-label">Output:</div>
                      <pre className="tool-output-content">{toolOutput.content}</pre>
                    </div>
                  )}
                </>
              )}
            </div>
          );
        })}
      </div>
    );
  }

  return (
    <div className={`message ${isUser ? 'user' : 'assistant'}`}>
      <div className="message-content">
        <div className="message-role">
          {isUser ? 'You' : 'Assistant'}
        </div>
        <div className="message-text">
          {displayContent}
          {toolCallsDisplay}
        </div>
      </div>
    </div>
  );
};

export default Message;

