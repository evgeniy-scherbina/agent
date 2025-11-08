import './Sidebar.css';

const Sidebar = ({ conversations, selectedConversationId, onSelectConversation, onNewConversation, onDeleteConversation }) => {

  const getConversationTitle = (conv) => {
    if (conv.messages && conv.messages.length > 0) {
      const firstUserMessage = conv.messages.find(msg => msg.role === 'user');
      if (firstUserMessage) {
        return firstUserMessage.content.substring(0, 30) + (firstUserMessage.content.length > 30 ? '...' : '');
      }
    }
    return 'New Chat';
  };

  return (
    <div className="sidebar">
      <div className="sidebar-header">
        <button className="new-chat-btn" onClick={onNewConversation}>
          <span>+</span> New Chat
        </button>
      </div>
      
      <div className="conversations-list">
        {conversations.length === 0 ? (
          <div className="empty-state">No conversations yet</div>
        ) : (
          conversations.map((conv) => (
            <div
              key={conv.id}
              className={`conversation-item ${selectedConversationId === conv.id ? 'active' : ''}`}
              onClick={() => onSelectConversation(conv.id)}
            >
              <span className="conversation-title">{getConversationTitle(conv)}</span>
              <button
                className="delete-btn"
                onClick={(e) => {
                  e.stopPropagation();
                  onDeleteConversation(conv.id);
                }}
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

export default Sidebar;

