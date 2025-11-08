import './Message.css';

const Message = ({ message }) => {
  const isUser = message.role === 'user';
  const isAssistant = message.role === 'assistant';

  // Handle assistant messages that are calling tools
  let displayContent = message.content;
  if (isAssistant && !displayContent && message.tool_calls && message.tool_calls.length > 0) {
    if (message.tool_calls.length === 1) {
      displayContent = `calling tool: ${message.tool_calls[0].name || 'tool'}`;
    } else {
      displayContent = `calling ${message.tool_calls.length} tools`;
    }
  }

  return (
    <div className={`message ${isUser ? 'user' : 'assistant'}`}>
      <div className="message-content">
        <div className="message-role">
          {isUser ? 'You' : 'Assistant'}
        </div>
        <div className="message-text">{displayContent}</div>
      </div>
    </div>
  );
};

export default Message;

