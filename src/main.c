// IO stuff, output, reading files, etc.
#include <stdio.h>

// atoi for arguments parsing, qsort, malloc, and more?
#include <stdlib.h>

// String utilities like strcmp, strcpy, etc
#include <string.h>

// git bindings
#include <git2.h>

// JSON library
#include <jansson.h>

// POSIX API. For gwtcwd.
#include <unistd.h>

// To have a reference to MAXPATHLEN
#include <sys/param.h>

// Uncomment to compile with debugging output turned on.
// #define DEBUG 1

//Define bool type
typedef int bool;
enum { false, true };

// Globals are evil, but this is a short-live program that will never change
// context while it's running.
git_repository *gb_repo;
json_t *gb_json;
char *gb_cache_path;
int *ahead_filter;

char *RED = "\e[0;31m";
char *YELLOW = "\e[0;33m";
char *GREEN = "\e[0;32m";

void gb_git_check_return(int rc, char *msg) {
  if (rc != 0) {
    fprintf(stderr, "%s. Code: %d\n", msg, rc);
    exit(1);
  }
}

int gb_rev_count(char *one, char *two) {
  if ( strcmp(one, two) == 0) {
    return 0;
  }

  char *range = malloc( (strlen(one) + strlen(two) + 3) * sizeof(char));
  sprintf(range, "%s..%s", one, two);

  // Return value read from cache if found.
  json_t *object = json_object_get(gb_json, range);
  if (json_is_number(object)) {
    return json_integer_value(object);
  }

  // Find value.
  int rc;
  git_revwalk *walker;
  git_oid next_commit;
  int count = 0;

  rc = git_revwalk_new(&walker, gb_repo);
  gb_git_check_return(rc, "new revwalk");

  rc = git_revwalk_push_range(walker, range);
  gb_git_check_return(rc, range);

  while (!git_revwalk_next(&next_commit, walker)) {
    count++;
  }

  // Cache count in JSON tree.
  json_object_set(gb_json, range, json_integer(count));

  git_revwalk_free(walker);
  return count;
}


typedef struct gb_comparison {
  char tip[41];
  char master_tip[41];
  git_oid tip_oid;
  char name[200];
  char reference_name[200];
  long timestamp;
  int ahead;
  int behind;
  int is_head;
} gb_comparison;

void gb_comparison_new(git_reference *ref, gb_comparison *comp) {
  int rc;

  memset(comp->tip, '\0', 41);
  memset(comp->master_tip, '\0', 41);

  // Find branch name.
  const char *name;
  git_branch_name(&name, ref);
  memset(comp->name, '\0', 200);
  strcpy(comp->name, name);

  // Assign full reference name.
  memset(comp->reference_name, '\0', 200);
  strcat(comp->reference_name, "refs/heads/");
  strcat(comp->reference_name, comp->name);

  // Choose color of output.
  comp->is_head = git_branch_is_head(ref);

  // Find tip oid.
  rc = git_reference_name_to_id(&comp->tip_oid, gb_repo, comp->reference_name);
  gb_git_check_return(rc, "Can't find branch tip id.");
  git_oid_tostr(comp->tip, 41, &comp->tip_oid);

  // Keep reference to master_tip that we're comparing to.
  git_oid master_oid;
  rc = git_reference_name_to_id(&master_oid, gb_repo, "refs/heads/master");
  gb_git_check_return(rc, "Can't find branch tip id.");
  git_oid_tostr(comp->master_tip, 41, &master_oid);

  // Find commit based on tip oid.
  git_commit *commit;
  git_commit_lookup(&commit, gb_repo, &(comp->tip_oid));

  // Assign timestamp.
  comp->timestamp = git_commit_time(commit);

  comp->ahead = 0;
  comp->behind = 0;
}


int gb_comparison_asc_timestamp_sort(const void *a, const void *b) {
  gb_comparison *x = *(gb_comparison **) a;
  gb_comparison *y = *(gb_comparison **) b;

  if (x->timestamp < y->timestamp) return  1;
  if (x->timestamp > y->timestamp) return -1;

  return 0;
}

int gb_comparison_desc_timestamp_sort(const void *a, const void *b) {
  gb_comparison *x = *(gb_comparison **) a;
  gb_comparison *y = *(gb_comparison **) b;

  if (x->timestamp > y->timestamp) return  1;
  if (x->timestamp < y->timestamp) return -1;

  return 0;
}


void gb_comparison_execute(gb_comparison *comp) {
  comp->ahead  = gb_rev_count(comp->master_tip, comp->tip);
  comp->behind = gb_rev_count(comp->tip, comp->master_tip);
}


// Returns a pointer to the color constant based on some very basic rules.
char* gb_output_color(gb_comparison *comp) {
  if ( !isatty(fileno(stdout)) ) return "";

  time_t rawtime = comp->timestamp;
  time_t now = time(0);
  int one_week = (14 * 24 * 60 * 60);

  if (comp->is_head) {
    return GREEN;
  } else if ( rawtime > (now - one_week) ) {
    return YELLOW;
  } else {
    return RED;
  }
}

void gb_comparison_print(gb_comparison *comp) {
  char formatted_time[80];
  time_t rawtime = comp->timestamp;
  struct tm * timeinfo = localtime(&rawtime);
  strftime(formatted_time, 80, "%F %H:%M%p", timeinfo);

  printf("%s%s | %-40.40s | behind: %4d | ahead: %4d\n",
         gb_output_color(comp),
         formatted_time,
         comp->name,
         comp->behind,
         comp->ahead);
}

bool gb_is_filtered_branch(gb_comparison *comp) {
  if (ahead_filter == NULL || comp->ahead == *ahead_filter) {
    return true;
  } else {
    return false;
  }
}




void print_last_branches() {
  gb_comparison **comps = malloc( sizeof(gb_comparison*) );

  int branch_count = 0;

  git_branch_iterator *iter;
	int rc;

	rc = git_branch_iterator_new(&iter, gb_repo, GIT_BRANCH_LOCAL);
  gb_git_check_return(rc, "Can't iterate over branches.");

	git_reference *ref = NULL;
	git_branch_t type;

	while (!(rc = git_branch_next(&ref, &type, iter))) {
    comps = (gb_comparison**) realloc(comps, (branch_count+1) * sizeof(gb_comparison*));
    gb_comparison *comp = malloc(sizeof(gb_comparison));
    gb_comparison_new(ref, comp);
    comps[branch_count] = comp;
    branch_count++;
	}

  qsort(comps, branch_count, sizeof(*comps), gb_comparison_desc_timestamp_sort);

  for (int i = 0; i < branch_count; i++) {
    gb_comparison_execute(comps[i]);
    if (gb_is_filtered_branch(comps[i])) {
      gb_comparison_print(comps[i]);
    }
  }

  git_branch_iterator_free(iter);

}

void gb_cache_load() {
  json_error_t error;

  gb_json = json_load_file(gb_cache_path, 0, &error);

  if (!gb_json) {
    #ifdef DEBUG
    if (error.line > 0) {
      fprintf(stderr, "error: on line %d: %s\n", error.line, error.text);
    } else {
      fprintf(stderr, "error: %s\n", error.text);
    }
    #endif

    // If JSON load failed (file does not exist, syntax error, etc), simply
    // proceed forward with an empty json_object.
    gb_json = json_object();
  }
}

void gb_cache_dump() {
  json_dump_file(gb_json, gb_cache_path, 0);
}


git_repository* gb_git_repo_new() {
  git_repository *repo;
  char cwd[MAXPATHLEN];

  if (getcwd(cwd, sizeof(cwd)) == NULL) {
    fprintf(stderr, "fatal: Could not get current working directory.\n");
    exit(1);
  }

  int rc = git_repository_open_ext(&repo, cwd, 0, NULL);
  if (rc == GIT_ENOTFOUND) {
    fprintf(stderr, "fatal: Not a git repository (or any of the parent directories): .git\n");
    exit(1);
  }

  gb_git_check_return(rc, "opening repository");

  return repo;
}




int main(int argc, char **args) {
  //Parse arguments.
  int opt;
  int ahead_option;
  while ((opt = getopt(argc, args, "a:")) != -1) {
    switch(opt){
    case 'a':
      ahead_option = atoi(optarg);
      ahead_filter = &ahead_option;
      break;
    default:
      printf("option not found");
    }
  }

  // First thing we do is init/load the globals.
  gb_repo = gb_git_repo_new();

  gb_cache_path = malloc(MAXPATHLEN * sizeof(char));
  sprintf(gb_cache_path, "%sgb_cache.json", git_repository_path(gb_repo));

  gb_cache_load();

  // Program run.
	print_last_branches();

  gb_cache_dump();

  return 0;
}
