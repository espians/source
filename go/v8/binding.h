#ifdef __cplusplus
extern "C" {
#endif

struct worker_s;
typedef struct worker_s worker;

void v8_init();

void worker_dispose(worker* w);

worker* worker_init(int id, int enable_print);

const char* worker_last_exception(worker* w);

int worker_load_module(worker* w, char* url_s);
int worker_load_script(worker* w, char* name_s, char* source_s);

int worker_send(worker* w, const char* msg);
const char* worker_send_sync(worker* w, const char* msg);

void worker_terminate_execution(worker* w);

const char* worker_version();

#ifdef __cplusplus
}  // extern "C"
#endif
