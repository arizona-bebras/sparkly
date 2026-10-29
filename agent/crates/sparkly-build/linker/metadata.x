/* The original GNU note and metric dictionary stay in the ELF, not flash.
 * The runtime ID copy is an ordinary .rodata static placed by the HAL. */
SECTIONS {
  .note.gnu.build-id 0 (INFO) : { KEEP(*(.note.gnu.build-id)) }
  .sparkly.metrics 0 (INFO) : {
    KEEP(*(.sparkly.metrics))
    KEEP(*(.sparkly.metrics.*))
  }
}
INSERT AFTER .comment;
