import React from 'react';
import { DemoCanvas } from './DemoCanvas';

export const DemoPage: React.FC = () => {
  return (
    <OuterContainer>
      <span
        style={{
          alignSelf: 'center',
          fontSize: '24pt',
          fontWeight: 700,
          marginTop: '4px',
        }}
      >
        Hello Spanreed Client
      </span>
      <DemoCanvas />
    </OuterContainer>
  );
};

interface OuterContainerProps {
  children?: React.ReactNode;
}
const OuterContainer: React.FC<OuterContainerProps> = ({ children }) => {
  return (
    <div
      style={{
        border: '4px solid #fff',
        borderRadius: '8px',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      {children}
    </div>
  );
};
