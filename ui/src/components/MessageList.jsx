import Message from './Message';
import './MessageList.css';

const MessageList = ({ messages }) => {
  // Filter out tool messages - they're redundant as the assistant summarizes them
  const filteredMessages = messages.filter(msg => msg.role !== 'tool');
  
  return (
    <div className="message-list">
      {filteredMessages.map((message, index) => (
        <Message key={message.ID || index} message={message} />
      ))}
    </div>
  );
};

export default MessageList;

