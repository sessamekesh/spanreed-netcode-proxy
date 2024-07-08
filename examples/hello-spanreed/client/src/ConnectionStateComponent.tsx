import React from 'react';

interface ConnectionStateComponentProps {}

export const ConnectionStateComponent: React.FC<ConnectionStateComponentProps> =
  React.memo(() => {
    return <span>Connection state component</span>;
  });
