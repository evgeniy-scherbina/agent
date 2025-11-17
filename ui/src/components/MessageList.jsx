import Message from './Message';
import './MessageList.css';

const MessageList = ({ messages }) => {
  // Group tool messages with their corresponding assistant messages
  const processedMessages = [];
  const toolMessagesByCallId = {};
  
  // First pass: collect tool messages
  messages.forEach(msg => {
    if (msg.role === 'tool') {
      // Tool messages can have tool_call_id or TollCallID (Go struct field)
      const toolCallId = msg.tool_call_id || msg.TollCallID;
      if (toolCallId) {
        if (!toolMessagesByCallId[toolCallId]) {
          toolMessagesByCallId[toolCallId] = [];
        }
        toolMessagesByCallId[toolCallId].push(msg);
      }
    }
  });
  
  // Second pass: build message list with tool messages attached
  messages.forEach((message, index) => {
    if (message.role === 'tool') {
      // Tool messages are handled separately, attached to assistant messages
      return;
    }
    
    // Find tool messages for this assistant message
    let toolMessages = [];
    if (message.role === 'assistant' && message.tool_calls) {
      message.tool_calls.forEach(toolCall => {
        if (toolMessagesByCallId[toolCall.id]) {
          toolMessages.push(...toolMessagesByCallId[toolCall.id]);
        }
      });
    }
    
    processedMessages.push({
      message,
      toolMessages,
      index
    });
  });
  
  return (
    <div className="message-list">
      {processedMessages.map(({ message, toolMessages, index }) => (
        <Message 
          key={message.ID || index} 
          message={message} 
          toolMessages={toolMessages}
        />
      ))}
    </div>
  );
};

export default MessageList;

