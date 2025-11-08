import { useState, useRef, useEffect } from 'react';
import './MessageInput.css';

const MessageInput = ({ onSend, isLoading }) => {
  const [message, setMessage] = useState('');
  const textareaRef = useRef(null);

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (message.trim() && !isLoading) {
      const messageToSend = message.trim();
      setMessage('');
      await onSend(messageToSend);
    }
  };

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
  };

  useEffect(() => {
    if (textareaRef.current) {
      textareaRef.current.style.height = 'auto';
      textareaRef.current.style.height = `${textareaRef.current.scrollHeight}px`;
    }
  }, [message]);

  return (
    <div className="message-input-container">
      <form onSubmit={handleSubmit} className="message-input-form">
        <textarea
          ref={textareaRef}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          placeholder="Message Agent..."
          rows={1}
          disabled={isLoading}
          className="message-input"
        />
        <button
          type="submit"
          disabled={!message.trim() || isLoading}
          className="send-button"
        >
          {isLoading ? '...' : '→'}
        </button>
      </form>
    </div>
  );
};

export default MessageInput;

