import Message from './Message';
import './MessageList.css';

const MessageList = ({ messages }) => {
  return (
    <div className="message-list">
      {messages.map((message, index) => (
        <Message key={message.ID || index} message={message} />
      ))}
    </div>
  );
};

export default MessageList;

