import { useState, useRef, useEffect } from 'react';
import MessageList from './MessageList';
import MessageInput from './MessageInput';
import './ChatInterface.css';

const ChatInterface = ({ conversationId, messages, onSendMessage, isLoading }) => {
  const messagesEndRef = useRef(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSend = async (message) => {
    await onSendMessage(message, conversationId);
  };

  return (
    <div className="chat-interface">
      <div className="chat-messages-container">
        {messages.length === 0 ? (
          <div className="welcome-message">
            <h1>Agent Chat</h1>
            <p>Start a conversation by typing a message below.</p>
          </div>
        ) : (
          <MessageList messages={messages} />
        )}
        <div ref={messagesEndRef} />
      </div>
      <MessageInput onSend={handleSend} isLoading={isLoading} />
    </div>
  );
};

export default ChatInterface;

