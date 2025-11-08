import './Message.css';

const Message = ({ message }) => {
  const isUser = message.role === 'user';
  const isAssistant = message.role === 'assistant';

  return (
    <div className={`message ${isUser ? 'user' : 'assistant'}`}>
      <div className="message-content">
        <div className="message-role">
          {isUser ? 'You' : 'Assistant'}
        </div>
        <div className="message-text">{message.content}</div>
      </div>
    </div>
  );
};

export default Message;

