import React, { useState } from "react";
import { LogLevel } from "./log";
import { Console } from "./Console";

export const BenchmarkPage: React.FC = () => {
  const [lines, setLines] = useState<
    Array<{ msg: string; logLevel: LogLevel }>
  >([]);

  return (
    <OuterContainer>
      <span
        style={{
          alignSelf: "center",
          fontSize: "24pt",
          fontWeight: 700,
          marginTop: "4px",
        }}
      >
        Hello Spanreed Client
      </span>
      <Console lines={lines} />
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
        border: "4px solid #fff",
        borderRadius: "8px",
        display: "flex",
        flexDirection: "column",
        maxWidth: "800px",
        marginLeft: "auto",
        marginRight: "auto",
      }}
    >
      {children}
    </div>
  );
};
