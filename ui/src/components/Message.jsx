import './Message.css';

const Message = ({ message }) => {
  const isUser = message.role === 'user';
  const isAssistant = message.role === 'assistant';

  // Handle assistant messages that are calling tools
  let displayContent = message.content;
  let toolCallsDisplay = null;
  
  if (isAssistant && !displayContent && message.tool_calls && message.tool_calls.length > 0) {
    displayContent = `calling ${message.tool_calls.length} ${message.tool_calls.length === 1 ? 'tool' : 'tools'}:`;
    
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
          
          return (
            <div key={toolCall.id || index} className="tool-call-item">
              <div className="tool-call-name">
                <strong>{toolCall.name || 'tool'}</strong>
              </div>
              <div className="tool-call-params">
                {Object.entries(parsedArgs).map(([key, value]) => (
                  <div key={key} className="tool-call-param">
                    <span className="param-key">{key}:</span>
                    <span className="param-value">{typeof value === 'string' ? value : JSON.stringify(value)}</span>
                  </div>
                ))}
              </div>
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

